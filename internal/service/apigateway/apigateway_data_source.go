package apigateway

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/apigateway"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/idanhaitner/terraform-provider-noname/internal/conns"
)

func DataSourceApiGateway() *schema.Resource {
	return &schema.Resource{
		Description: `Use this data source to get the access to the effective
		Account ID, User ID, ARN and EKS Role ARN in which Terraform is authorized.`,
		Read: dataSourceApiGatewayRed,
		Schema: map[string]*schema.Schema{
			"rest_api_id": {
				Description: `AWS Account ID number of the account that owns or contains the calling entity.`,
				Type:        schema.TypeString,
				Required:    true,
			},
			"stages": {
				Description: `List of stages of the API`,
				Type:        schema.TypeList,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Computed:    true,
			},
		},
	}
}

func dataSourceApiGatewayRed(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*conns.AWSClient).APIGatewayConn
	restApiId := d.Get("rest_api_id").(string)
	res, err := client.GetStages(&apigateway.GetStagesInput{RestApiId: &restApiId})
	if err != nil {
		return fmt.Errorf("getting REST API Stages: %w", err)
	}
	stages := []string{}
	for _, stage := range res.Item {
		stages = append(stages, *stage.StageName)
	}

	d.SetId(*aws.String(restApiId))
	d.Set("rest_api_id", aws.String(restApiId))
	d.Set("stages", aws.StringSlice(stages))
	return nil
}
