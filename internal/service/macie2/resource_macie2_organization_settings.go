package macie2

import (
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/macie2"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/idanhaitner/terraform-provider-noname/internal/conns"
	"github.com/idanhaitner/terraform-provider-noname/internal/flex"
)

func ResourceAwsUtilsMacie2OrganizationSettings() *schema.Resource {
	return &schema.Resource{
		Description: `Enables a list of accounts as Macie2 member accounts in an existing AWS Organization.

Designating an account as the Macie2 Administrator account in an AWS Organization can optionally enable all
newly created accounts and accounts that join the organization after the setting is enabled, however, it does not
enable existing accounts. Use this resource to enable a list of existing accounts.`,
		Create:        resourceAwsMacie2OrganizationSettingsCreate,
		Read:          resourceAwsMacie2OrganizationSettingsRead,
		Update:        resourceAwsMacie2OrganizationSettingsUpdate,
		Delete:        resourceAwsMacie2OrganizationSettingsDelete,
		SchemaVersion: 1,
		Schema: map[string]*schema.Schema{
			"id": {
				Description: "The ID of this resource.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"member_accounts": {
				Description: "A list of AWS Organization member accounts to associate with the Macie2 Administrator account.",
				Type:        schema.TypeSet,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Set:         schema.HashString,
				Required:    true,
			},
		},
	}
}

func getMemberAccounts(d *schema.ResourceData) []string {
	memberAccounts := flex.ExpandStringSliceofPointers(flex.ExpandStringSet(d.Get("member_accounts").(*schema.Set)))
	return memberAccounts
}

func getMemberAccountsFromAws(conn *macie2.Macie2) ([]string, error) {
	memberAccounts, err := conn.ListMembers(&macie2.ListMembersInput{})
	if err != nil {
		return nil, fmt.Errorf("error reading macie2 organization members: %s", err)
	}

	var memberAccountIDs []string

	for i := range memberAccounts.Members {
		memberAccountIDs = append(memberAccountIDs, *memberAccounts.Members[i].AccountId)
	}
	return memberAccountIDs, nil
}

func resourceAwsMacie2OrganizationSettingsCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).Macie2Conn
	memberAccounts := getMemberAccounts(d)

	if err := addMacie2OrganizationMembers(conn, memberAccounts); err != nil {
		return err
	}

	d.SetId(uuid.New().String())

	return resourceAwsMacie2OrganizationSettingsRead(d, meta)
}

func resourceAwsMacie2OrganizationSettingsRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).Macie2Conn
	memberAccounts, err := getMemberAccountsFromAws(conn)
	if err != nil {
		return err
	}

	d.Set("member_accounts", memberAccounts)

	return nil
}

func resourceAwsMacie2OrganizationSettingsUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).Macie2Conn
	currentMemberAccounts, err := getMemberAccountsFromAws(conn)
	if err != nil {
		return err
	}

	desiredMemberAccounts := flex.ExpandStringSliceofPointers(flex.ExpandStringSet(d.Get("member_accounts").(*schema.Set)))

	membersToAdd := flex.Diff(desiredMemberAccounts, currentMemberAccounts)
	if len(membersToAdd) > 0 {
		if err := addMacie2OrganizationMembers(conn, membersToAdd); err != nil {
			return fmt.Errorf("error setting macie2 organization members: %s", err)
		}
	}

	membersToRemove := flex.Diff(currentMemberAccounts, desiredMemberAccounts)
	if len(membersToRemove) > 0 {
		if err := removeMacie2OrganizationMembers(conn, membersToRemove); err != nil {
			return fmt.Errorf("error removing macie2 organization members: %s", err)
		}
	}
	return nil
}

func resourceAwsMacie2OrganizationSettingsDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).Macie2Conn
	membersToRemove := getMemberAccounts(d)

	if err := removeMacie2OrganizationMembers(conn, membersToRemove); err != nil {
		return fmt.Errorf("error removing macie2 organization members: %s", err)
	}
	return nil
}

func makeMacie2AccountDetails(accounts []string) []*macie2.AccountDetail {
	accountDetails := make([]*macie2.AccountDetail, 0)
	for i := range accounts {
		accountDetails = append(accountDetails, &macie2.AccountDetail{
			AccountId: aws.String(accounts[i]),
			Email:     aws.String("notused2join@awsorganization.com"),
		})
	}
	return accountDetails
}

func makeMacie2AccountIDs(accounts []string) []*string {
	accountIDs := make([]*string, 0)
	for i := range accounts {
		accountIDs = append(accountIDs, aws.String(accounts[i]))
	}
	return accountIDs
}

func addMacie2OrganizationMembers(conn *macie2.Macie2, memberAccounts []string) error {
	if len(memberAccounts) > 0 {
		accountDetails := makeMacie2AccountDetails(memberAccounts)

		for i := range accountDetails {
			createMemberInput := &macie2.CreateMemberInput{
				Account: accountDetails[i],
			}

			if _, err := conn.CreateMember(createMemberInput); err != nil {
				return fmt.Errorf("error designating macie2 administrator account members: %s", err)
			}
		}
	}
	return nil
}

func removeMacie2OrganizationMembers(conn *macie2.Macie2, memberAccounts []string) error {
	accountIDs := makeMacie2AccountIDs(memberAccounts)
	if len(memberAccounts) > 0 {

		for i := range memberAccounts {
			disassociateMemberInput := &macie2.DisassociateMemberInput{
				Id: accountIDs[i],
			}

			deleteMemberInput := &macie2.DeleteMemberInput{
				Id: accountIDs[i],
			}

			if _, err := conn.DisassociateMember(disassociateMemberInput); err != nil {
				if err != nil {
					return fmt.Errorf("error disassociating macie2 administrator account member: %s", err)
				}
			}

			if _, err := conn.DeleteMember(deleteMemberInput); err != nil {
				log.Printf("[WARN] Error deleting macie2 administrator account member: %s", err.Error())
				if strings.Contains(err.Error(), "specified account is not associated with your account") {
					log.Printf("[WARN] The specified member account (%s) isn't associated with the delegated administrator account", *disassociateMemberInput.Id)
				} else {
					return fmt.Errorf("error removing macie2 administrator account members: %s", err)
				}
			}
		}
	}
	return nil
}
