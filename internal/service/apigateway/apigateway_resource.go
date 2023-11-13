package apigateway

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go/service/apigateway"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/idanhaitner/terraform-provider-noname/internal/conns"
)

func ResourceApiGateway() *schema.Resource {
	return &schema.Resource{
		Description: `Use this data source to get the access to the effective
		Account ID, User ID, ARN and EKS Role ARN in which Terraform is authorized.`,
		Read:   resourceApiGatewayRead,
		Create: resourceApiGatewayCreate,
		Delete: resourceApiGatewayDelete,
		Update: resourceApiGatewayUpdate,
		Schema: map[string]*schema.Schema{
			"rest_api_id": {
				Description: `AWS Account ID number of the account that owns or contains the calling entity.`,
				Type:        schema.TypeString,
				Required:    true,
			},
			"stage_name": {
				Description: `List of stages of the API`,
				Type:        schema.TypeString,
				Required:    true,
			},
			"description": {
				Description: `Description of the stage`,
				Type:        schema.TypeString,
				Required:    true,
			},
			"current_description": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceApiGatewayRead(d *schema.ResourceData, meta interface{}) error {
	restApiId := d.Get("rest_api_id").(string)
	stageName := d.Get("stage_name").(string)
	description := d.Get("description").(string)
	d.SetId(restApiId + "_" + stageName)
	d.Set("stage_name", stageName)
	d.Set("rest_api_id", restApiId)
	d.Set("description", description)
	return nil
}

func saveCurrentDescriptionState(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).APIGatewayConn
	restApiId := d.Get("rest_api_id").(string)
	stageName := d.Get("stage_name").(string)
	res, err := conn.GetStage(&apigateway.GetStageInput{
		RestApiId: &restApiId,
		StageName: &stageName,
	})
	if err != nil {
		return err
	}
	d.Set("current_description", res.Description)
	return nil
}

func resourceApiGatewayCreate(d *schema.ResourceData, meta interface{}) error {
	saveCurrentDescriptionState(d, meta)
	conn := meta.(*conns.AWSClient).APIGatewayConn
	restApiId := d.Get("rest_api_id").(string)
	stageName := d.Get("stage_name").(string)
	description := d.Get("description").(string)
	res, err := updateStageDescription(conn, restApiId, stageName, description)
	if err != nil {
		return err
	}

	d.Set("descirption", res.Description)
	return resourceApiGatewayRead(d, meta)
}

func updateStageDescription(conn *apigateway.APIGateway, restApiId string, stageName string, description string) (*apigateway.Stage, error) {
	res, err := conn.UpdateStage(&apigateway.UpdateStageInput{
		RestApiId: &restApiId,
		StageName: &stageName,
		PatchOperations: []*apigateway.PatchOperation{
			{
				Op:    aws.String("replace"),
				Path:  aws.String("/description"),
				Value: aws.String(description),
			},
		},
	})
	return res, err
}

func resourceApiGatewayUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).APIGatewayConn
	if d.HasChanges("rest_api_id") || d.HasChange("stage_name") || d.HasChange("description") {
		restApiId := d.Get("rest_api_id").(string)
		stageName := d.Get("stage_name").(string)
		description := d.Get("description").(string)
		_, err := updateStageDescription(conn, restApiId, stageName, description)
		return err
	}
	return resourceApiGatewayRead(d, meta)
}

func resourceApiGatewayDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).APIGatewayConn
	restApiId := d.Get("rest_api_id").(string)
	stageName := d.Get("stage_name").(string)
	description := d.Get("current_description").(string)
	_, err := updateStageDescription(conn, restApiId, stageName, description)
	if err != nil {
		return err
	}
	return nil
}
