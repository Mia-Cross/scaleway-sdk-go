package instance

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/scaleway/scaleway-sdk-go/internal/testhelpers"
	"github.com/scaleway/scaleway-sdk-go/internal/testhelpers/httprecorder"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

func TestAPI_UpdateSecurityGroup(t *testing.T) {
	client, r, err := httprecorder.CreateRecordedScwClient("security-group-test")
	testhelpers.AssertNoError(t, err)
	defer func() {
		testhelpers.AssertNoError(t, r.Stop()) // Make sure recorder is stopped once done with it
	}()

	instanceAPI := NewAPI(client)

	createResponse, err := instanceAPI.CreateSecurityGroup(&CreateSecurityGroupRequest{
		Name:                  "name",
		Description:           "description",
		Stateful:              true,
		InboundDefaultPolicy:  SecurityGroupPolicyAccept,
		OutboundDefaultPolicy: SecurityGroupPolicyDrop,
	})

	testhelpers.AssertNoError(t, err)

	accept := SecurityGroupPolicyAccept
	drop := SecurityGroupPolicyDrop

	updateResponse, err := instanceAPI.UpdateSecurityGroup(&UpdateSecurityGroupRequest{
		SecurityGroupID:       createResponse.SecurityGroup.ID,
		Name:                  scw.StringPtr("new_name"),
		Description:           scw.StringPtr("new_description"),
		Stateful:              scw.BoolPtr(false),
		InboundDefaultPolicy:  drop,
		OutboundDefaultPolicy: accept,
		// Keep false here, switch it to true is too dangerous for the one who update the test cassette.
		ProjectDefault: scw.BoolPtr(false),
		Tags:           scw.StringsPtr([]string{"foo", "bar"}),
	})

	testhelpers.AssertNoError(t, err)
	testhelpers.Equals(t, "new_name", updateResponse.SecurityGroup.Name)
	testhelpers.Equals(t, "new_description", updateResponse.SecurityGroup.Description)
	testhelpers.Equals(t, SecurityGroupPolicyDrop, updateResponse.SecurityGroup.InboundDefaultPolicy)
	testhelpers.Equals(t, SecurityGroupPolicyAccept, updateResponse.SecurityGroup.OutboundDefaultPolicy)
	testhelpers.Equals(t, false, updateResponse.SecurityGroup.Stateful)
	testhelpers.Equals(t, false, updateResponse.SecurityGroup.ProjectDefault)
	testhelpers.Equals(t, false, *updateResponse.SecurityGroup.OrganizationDefault)
	testhelpers.Equals(t, []string{"foo", "bar"}, updateResponse.SecurityGroup.Tags)

	err = waitForSecurityGroup(client, createResponse.SecurityGroup.Zone, createResponse.SecurityGroup.ID, 10)
	testhelpers.AssertNoError(t, err)

	err = instanceAPI.DeleteSecurityGroup(&DeleteSecurityGroupRequest{
		SecurityGroupID: createResponse.SecurityGroup.ID,
		Zone:            createResponse.SecurityGroup.Zone,
	})
	testhelpers.AssertNoError(t, err)
}

func TestAPI_UpdateSecurityGroupRule(t *testing.T) {
	client, r, err := httprecorder.CreateRecordedScwClient("security-group-rule-test")
	testhelpers.AssertNoError(t, err)
	defer func() {
		testhelpers.AssertNoError(t, r.Stop()) // Make sure recorder is stopped once done with it
	}()

	instanceAPI := NewAPI(client)

	bootstrap := func(t *testing.T) (*SecurityGroup, *SecurityGroupRule, func()) {
		t.Helper()
		createSecurityGroupResponse, err := instanceAPI.CreateSecurityGroup(&CreateSecurityGroupRequest{
			Name:                  "name",
			Description:           "description",
			Stateful:              true,
			InboundDefaultPolicy:  SecurityGroupPolicyAccept,
			OutboundDefaultPolicy: SecurityGroupPolicyDrop,
		})

		testhelpers.AssertNoError(t, err)
		_, ipNet, _ := net.ParseCIDR("8.8.8.8/32")
		createRuleResponse, err := instanceAPI.CreateSecurityGroupRule(&CreateSecurityGroupRuleRequest{
			SecurityGroupID: createSecurityGroupResponse.SecurityGroup.ID,
			Direction:       SecurityGroupRuleDirectionInbound,
			Protocol:        SecurityGroupRuleProtocolTCP,
			DestPortFrom:    scw.Uint32Ptr(1),
			DestPortTo:      scw.Uint32Ptr(1024),
			IPRange:         scw.IPNet{IPNet: *ipNet},
			Action:          SecurityGroupRuleActionAccept,
			Position:        1,
		})
		testhelpers.AssertNoError(t, err)

		return createSecurityGroupResponse.SecurityGroup, createRuleResponse.Rule, func() {
			zone := createSecurityGroupResponse.SecurityGroup.Zone
			sgID := createSecurityGroupResponse.SecurityGroup.ID

			err = waitForSecurityGroup(client, zone, sgID, 10)
			testhelpers.AssertNoError(t, err)

			err = instanceAPI.DeleteSecurityGroup(&DeleteSecurityGroupRequest{
				SecurityGroupID: sgID,
				Zone:            zone,
			})
			testhelpers.AssertNoError(t, err)
		}
	}

	t.Run("Simple update", func(t *testing.T) {
		group, rule, cleanUp := bootstrap(t)
		defer cleanUp()

		_, ipNet, _ := net.ParseCIDR("1.1.1.1/32")
		updateResponse, err := instanceAPI.UpdateSecurityGroupRule(&UpdateSecurityGroupRuleRequest{
			SecurityGroupID:     group.ID,
			SecurityGroupRuleID: rule.ID,
			Action:              SecurityGroupRuleActionDrop,
			IPRange:             &scw.IPNet{IPNet: *ipNet},
			DestPortFrom:        scw.Uint32Ptr(1),
			DestPortTo:          scw.Uint32Ptr(2048),
			Protocol:            SecurityGroupRuleProtocolUDP,
			Direction:           SecurityGroupRuleDirectionOutbound,
		})

		testhelpers.AssertNoError(t, err)
		testhelpers.Equals(t, SecurityGroupRuleActionDrop, updateResponse.Rule.Action)
		testhelpers.Equals(t, scw.IPNet{IPNet: net.IPNet{IP: net.IPv4(0x1, 0x1, 0x1, 0x1), Mask: net.IPMask{0xff, 0xff, 0xff, 0xff}}}, updateResponse.Rule.IPRange)
		testhelpers.Equals(t, scw.Uint32Ptr(1), updateResponse.Rule.DestPortFrom)
		testhelpers.Equals(t, scw.Uint32Ptr(2048), updateResponse.Rule.DestPortTo)
		testhelpers.Equals(t, SecurityGroupRuleProtocolUDP, updateResponse.Rule.Protocol)
		testhelpers.Equals(t, SecurityGroupRuleDirectionOutbound, updateResponse.Rule.Direction)
	})

	t.Run("From a port range to a single port", func(t *testing.T) {
		group, rule, cleanUp := bootstrap(t)
		defer cleanUp()

		_, ipNet, _ := net.ParseCIDR("1.1.1.1/32")
		updateResponse, err := instanceAPI.UpdateSecurityGroupRule(&UpdateSecurityGroupRuleRequest{
			SecurityGroupID:     group.ID,
			SecurityGroupRuleID: rule.ID,
			Action:              SecurityGroupRuleActionDrop,
			IPRange:             &scw.IPNet{IPNet: *ipNet},
			DestPortFrom:        scw.Uint32Ptr(22),
			DestPortTo:          scw.Uint32Ptr(22),
			Protocol:            SecurityGroupRuleProtocolUDP,
			Direction:           SecurityGroupRuleDirectionOutbound,
		})

		testhelpers.AssertNoError(t, err)
		testhelpers.Equals(t, SecurityGroupRuleActionDrop, updateResponse.Rule.Action)
		testhelpers.Equals(t, scw.IPNet{IPNet: net.IPNet{IP: net.IPv4(0x1, 0x1, 0x1, 0x1), Mask: net.IPMask{0xff, 0xff, 0xff, 0xff}}}, updateResponse.Rule.IPRange)
		testhelpers.Equals(t, uint32(22), *updateResponse.Rule.DestPortFrom)
		testhelpers.Equals(t, (*uint32)(nil), updateResponse.Rule.DestPortTo)
		testhelpers.Equals(t, SecurityGroupRuleProtocolUDP, updateResponse.Rule.Protocol)
		testhelpers.Equals(t, SecurityGroupRuleDirectionOutbound, updateResponse.Rule.Direction)
	})

	t.Run("Switching to ICMP", func(t *testing.T) {
		group, rule, cleanUp := bootstrap(t)
		defer cleanUp()

		updateResponse, err := instanceAPI.UpdateSecurityGroupRule(&UpdateSecurityGroupRuleRequest{
			SecurityGroupID:     group.ID,
			SecurityGroupRuleID: rule.ID,
			Protocol:            SecurityGroupRuleProtocolICMP,
		})

		testhelpers.AssertNoError(t, err)
		testhelpers.Equals(t, SecurityGroupRuleActionAccept, updateResponse.Rule.Action)
		testhelpers.Equals(t, scw.IPNet{IPNet: net.IPNet{IP: net.IPv4(0x8, 0x8, 0x8, 0x8), Mask: net.IPMask{0xff, 0xff, 0xff, 0xff}}}, updateResponse.Rule.IPRange)
		testhelpers.Equals(t, (*uint32)(nil), updateResponse.Rule.DestPortFrom)
		testhelpers.Equals(t, (*uint32)(nil), updateResponse.Rule.DestPortTo)
		testhelpers.Equals(t, SecurityGroupRuleProtocolICMP, updateResponse.Rule.Protocol)
		testhelpers.Equals(t, SecurityGroupRuleDirectionInbound, updateResponse.Rule.Direction)
	})

	t.Run("Remove ports", func(t *testing.T) {
		group, rule, cleanUp := bootstrap(t)
		defer cleanUp()

		updateResponse, err := instanceAPI.UpdateSecurityGroupRule(&UpdateSecurityGroupRuleRequest{
			SecurityGroupID:     group.ID,
			SecurityGroupRuleID: rule.ID,
			DestPortFrom:        scw.Uint32Ptr(0),
			DestPortTo:          scw.Uint32Ptr(0),
		})

		testhelpers.AssertNoError(t, err)
		testhelpers.Equals(t, SecurityGroupRuleActionAccept, updateResponse.Rule.Action)
		testhelpers.Equals(t, scw.IPNet{IPNet: net.IPNet{IP: net.IPv4(0x8, 0x8, 0x8, 0x8), Mask: net.IPMask{0xff, 0xff, 0xff, 0xff}}}, updateResponse.Rule.IPRange)
		testhelpers.Equals(t, (*uint32)(nil), updateResponse.Rule.DestPortFrom)
		testhelpers.Equals(t, (*uint32)(nil), updateResponse.Rule.DestPortTo)
		testhelpers.Equals(t, SecurityGroupRuleProtocolTCP, updateResponse.Rule.Protocol)
		testhelpers.Equals(t, SecurityGroupRuleDirectionInbound, updateResponse.Rule.Direction)
	})
}

func waitForSecurityGroup(client *scw.Client, zone scw.Zone, sgID string, maxRetries int) error {
	instanceAPI := NewAPI(client)

	sg, err := instanceAPI.GetSecurityGroup(&GetSecurityGroupRequest{
		SecurityGroupID: sgID,
		Zone:            zone,
	})
	if err != nil {
		return err
	}

	switch sg.SecurityGroup.State {
	case SecurityGroupStateAvailable:
		return nil
	case SecurityGroupStateSyncingError:
		return nil
	case SecurityGroupStateSyncing:
		if maxRetries == 0 {
			return fmt.Errorf("max retries exceeded")
		}
		time.Sleep(100 * time.Millisecond)

		return waitForSecurityGroup(client, zone, sgID, maxRetries-1)
	default:
		return fmt.Errorf("unknown state: %s", sg.SecurityGroup.State)
	}
}
