package apigatewayintegration

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go/service/apigateway"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/idanhaitner/terraform-provider-noname/internal/conns"
)

type StageState struct {
	dataTraceEnabled         bool
	loggingLevel             string
	accessLogsFormat         string
	accessLogsDestinationArn string
}

func ResourceApiGatewayIntegration() *schema.Resource {
	return &schema.Resource{
		Description: `Use this data source to get the access to the effective
		Account ID, User ID, ARN and EKS Role ARN in which Terraform is authorized.`,
		Read:   resourceApiGatewayIntegrationRead,
		Create: resourceApiGatewayIntegrationCreate,
		Delete: resourceApiGatewayIntegrationDelete,
		Update: resourceApiGatewayIntegrationUpdate,
		Schema: map[string]*schema.Schema{
			"rest_api_ids": {
				Description: `AWS Account ID number of the account that owns or contains the calling entity.`,
				Type:        schema.TypeSet,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Required:    true,
			},
			"rest_api_states": {
				Description: `List of stages of the API`,
				Type:        schema.TypeMap,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Computed:    true,
			},
		},
	}
}

func resourceApiGatewayIntegrationRead(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func saveStagesStates(d *schema.ResourceData, conn *apigateway.APIGateway, restApiId string) map[string]interface{} {
	allStates := d.Get("rest_api_states").(map[string]interface{})
	res, _ := conn.GetStages(&apigateway.GetStagesInput{
		RestApiId: &restApiId,
	})

	for _, stage := range res.Item {
		identifier := fmt.Sprintf("%v-%v", restApiId, *stage.StageName)
		state := extractStageState(stage)
		allStates[identifier] = fmt.Sprintf("%v!%v!%v!%v",
			state.dataTraceEnabled,
			state.loggingLevel,
			state.accessLogsFormat,
			state.accessLogsDestinationArn,
		)
	}
	return allStates
}

func generateLogGroup(accountId string, region string, restApiId string, stageName string) string {
	return fmt.Sprintf("arn:aws:logs:%v:%v:log-group:API-Gateway-Execution-Logs_%v/%v", region, accountId, restApiId, stageName)
}

func extractStageState(stage *apigateway.Stage) StageState {
	format, destinationArn := getAccessLogsSettings(stage.AccessLogSettings)
	return StageState{
		dataTraceEnabled:         *stage.MethodSettings["*/*"].DataTraceEnabled,
		loggingLevel:             *stage.MethodSettings["*/*"].LoggingLevel,
		accessLogsFormat:         format,
		accessLogsDestinationArn: destinationArn,
	}
}

func resourceApiGatewayIntegrationCreate(d *schema.ResourceData, meta interface{}) error {

	restApiIds := d.Get("rest_api_ids").(*schema.Set)
	for _, restApiId := range restApiIds.List() {
		configureRestApi(meta, d, restApiId.(string))
	}
	d.SetId(uuid.New().String())
	return nil
}

func getAccessLogsSettings(settings *apigateway.AccessLogSettings) (string, string) {
	if settings == nil {
		return "NO", "NO"
	}
	return *settings.Format, *settings.DestinationArn
}

func configureRestApi(meta interface{}, d *schema.ResourceData, restApiId string) error {
	stsConn := meta.(*conns.AWSClient).STSConn
	res, _ := stsConn.GetCallerIdentity(&sts.GetCallerIdentityInput{})
	accountId := res.Account
	region := meta.(*conns.AWSClient).Session.Config.Region
	conn := meta.(*conns.AWSClient).APIGatewayConn
	allStates := saveStagesStates(d, conn, restApiId)
	d.Set("rest_api_states", allStates)
	apiRes, _ := conn.GetStages(&apigateway.GetStagesInput{
		RestApiId: &restApiId,
	})
	for _, stage := range apiRes.Item {
		conn.UpdateStage(&apigateway.UpdateStageInput{
			RestApiId: &restApiId,
			StageName: stage.StageName,
			PatchOperations: []*apigateway.PatchOperation{
				{
					Op:    aws.String("replace"),
					Path:  aws.String("/*/*/logging/loglevel"),
					Value: aws.String("INFO"),
				},
				{
					Op:    aws.String("replace"),
					Path:  aws.String("/*/*/logging/dataTrace"),
					Value: aws.String("true"),
				},
				{
					Op:    aws.String("replace"),
					Path:  aws.String("/accessLogSettings/format"),
					Value: aws.String(`{"requestId":"$context.requestId","ip":"$context.identity.sourceIp","caller":"$context.identity.caller","user":"$context.identity.user","requestTime":"$context.requestTime","httpMethod":"$context.httpMethod","path":"$context.path","status":"$context.status","protocol":"$context.protocol","responseLength":"$context.responseLength","domainName":"$context.domainName","accountId":"$context.accountId"}`),
				},
				{
					Op:    aws.String("replace"),
					Path:  aws.String("/accessLogSettings/destinationArn"),
					Value: aws.String(generateLogGroup(*accountId, *region, restApiId, *stage.StageName)),
				},
			},
		})
	}
	return nil
}

func resourceApiGatewayIntegrationUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).APIGatewayConn
	old, new := d.GetChange("rest_api_ids")
	if len(old.(*schema.Set).List()) > 0 {
		for _, restApiId := range old.(*schema.Set).List() {
			deconfigureRestApi(conn, d, restApiId.(string))
		}
	}
	if len(new.(*schema.Set).List()) > 0 {
		for _, restApiId := range new.(*schema.Set).List() {
			configureRestApi(meta, d, restApiId.(string))
		}
	}
	return nil
}

func deconfigureRestApi(conn *apigateway.APIGateway, d *schema.ResourceData, restApiId string) {
	allStates := d.Get("rest_api_states").(map[string]interface{})
	apiRes, _ := conn.GetStages(&apigateway.GetStagesInput{
		RestApiId: &restApiId,
	})
	for _, stage := range apiRes.Item {
		idenifier := fmt.Sprintf("%v-%v", restApiId, *stage.StageName)
		details := strings.Split(allStates[idenifier].(string), "!")
		traceEnabled := details[0]
		loggingLevel := details[1]
		accessLogsFormat := details[2]
		accessLogsDestinationArn := details[3]
		patchOperation := []*apigateway.PatchOperation{
			{
				Op:    aws.String("replace"),
				Path:  aws.String("/*/*/logging/loglevel"),
				Value: aws.String(loggingLevel),
			},
			{
				Op:    aws.String("replace"),
				Path:  aws.String("/*/*/logging/dataTrace"),
				Value: aws.String(traceEnabled),
			},
		}
		if accessLogsFormat == "N0" {
			patchOperation = append(patchOperation, &apigateway.PatchOperation{
				Op:   aws.String("remove"),
				Path: aws.String("/accessLogSettings"),
			})
		} else {
			patchOperation = append(patchOperation, []*apigateway.PatchOperation{
				{
					Op:    aws.String("replace"),
					Path:  aws.String("/accessLogSettings/destinationArn"),
					Value: aws.String(accessLogsDestinationArn),
				},
				{
					Op:    aws.String("replace"),
					Path:  aws.String("/accessLogSettings/format"),
					Value: aws.String(accessLogsFormat),
				},
			}...)
		}

		conn.UpdateStage(&apigateway.UpdateStageInput{
			RestApiId:       &restApiId,
			StageName:       stage.StageName,
			PatchOperations: patchOperation,
		})
		delete(allStates, idenifier)
	}
	d.Set("rest_api_states", allStates)
}

func resourceApiGatewayIntegrationDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).APIGatewayConn
	allStates := d.Get("rest_api_states").(map[string]interface{})
	for restApiId := range allStates {
		deconfigureRestApi(conn, d, restApiId)
	}
	d.SetId("")
	return nil
}
