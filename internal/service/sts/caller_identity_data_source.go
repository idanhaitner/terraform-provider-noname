package sts

import (
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/idanhaitner/terraform-provider-noname/internal/conns"
)

func DataSourceCallerIdentity() *schema.Resource {
	return &schema.Resource{
		Description: `Use this data source to get the access to the effective
		Account ID, User ID, ARN and EKS Role ARN in which Terraform is authorized.`,
		Read: dataSourceCallerIdentityRead,
		Schema: map[string]*schema.Schema{
			"account_id": {
				Description: `AWS Account ID number of the account that owns or contains the calling entity.`,
				Type:        schema.TypeString,
				Computed:    true,
			},

			"arn": {
				Description: `The AWS ARN associated with the calling entity.`,
				Type:        schema.TypeString,
				Computed:    true,
			},

			"eks_role_arn": {
				Description: `If the calling identity is an assumed role, this is the transformation of that assumed role ARN
				into something suitable for use in configuring EKS access. See
				[Add IAM principals to your Amazon EKS cluster](https://docs.aws.amazon.com/eks/latest/userguide/add-user-role.html#aws-auth-users).
				Otherwise, ` + "`null`" + `.`,
				Type:     schema.TypeString,
				Computed: true,
			},

			"id": {
				Description: `AWS Account ID number of the account that owns or contains the calling entity.`,
				Type:        schema.TypeString,
				Computed:    true,
			},

			"user_id": {
				Description: `Unique identifier of the calling entity.`,
				Type:        schema.TypeString,
				Computed:    true,
			},
		},
	}
}

// Take the ARN output of sts get caller identity and return the role
// example: arn:aws:sts::123456789012:assumed-role/role-name/role-session-name
// returns: arn:aws:iam::123456789012:role/role-name
func getIamRoleArnFromCallerRoleArn(sessionRole string) string {
	parts := strings.Split(sessionRole, "/")
	if len(parts) != 3 {
		return sessionRole
	}

	arnParts := strings.Split(parts[0], ":")
	if len(arnParts) != 6 {
		return sessionRole
	}

	accountId := arnParts[4]
	partition := arnParts[1]
	roleName := parts[1]

	return fmt.Sprintf("arn:%s:iam::%s:role/%s", partition, accountId, roleName)
}

func dataSourceCallerIdentityRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*conns.AWSClient).STSConn

	log.Printf("[DEBUG] Reading Caller Identity")
	res, err := client.GetCallerIdentity(&sts.GetCallerIdentityInput{})

	if err != nil {
		return fmt.Errorf("getting Caller Identity: %w", err)
	}

	log.Printf("[DEBUG] Received Caller Identity: %s", res)

	d.SetId(aws.StringValue(res.Account))
	d.Set("account_id", res.Account)
	d.Set("arn", res.Arn)
	d.Set("user_id", res.UserId)

	// If the caller identity is an assumed role, get the IAM role ARN and set it as the ARN
	if strings.HasPrefix(aws.StringValue(res.Arn), "arn:aws:sts") {
		d.Set("eks_role_arn", getIamRoleArnFromCallerRoleArn(aws.StringValue(res.Arn)))
	}

	return nil
}
