package kubevirt

import (
	"net"
	"testing"

	"github.com/insomniacslk/dhcp/dhcpv4"
	"github.com/stretchr/testify/assert"
	kubevirtv1 "kubevirt.io/api/core/v1"
)

func TestSetupKubevirt(t *testing.T) {
	// Test case 1: Valid argument
	handler, err := setupKubevirt("/Users/mbengtss/.kube/config")
	assert.NoError(t, err)
	assert.NotNil(t, handler)

	// Test case 2: Invalid argument
	handler, err = setupKubevirt()
	assert.Error(t, err)
	assert.Nil(t, handler)
}

func TestKubevirtHandler4(t *testing.T) {
	k := &KubevirtState{}
	req := &dhcpv4.DHCPv4{
		ClientHWAddr: net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
	}
	resp := &dhcpv4.DHCPv4{}

	// Test case 1: Machine instance found
	k.addKubevirtInstance(&KubevirtInstance{
		Name:      "test",
		Namespace: "test",
		Interfaces: []kubevirtv1.VirtualMachineInstanceNetworkInterface{
			{
				MAC: "00:11:22:33:44:55",
			},
		},
	})
	expectedResp := resp
	expectedContinue := false
	actualResp, actualContinue := k.kubevirtHandler4(req, resp)
	assert.Equal(t, expectedResp, actualResp)
	assert.Equal(t, expectedContinue, actualContinue)
}
