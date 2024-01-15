package kubevirt

import (
	"context"
	"net"
	"testing"

	"github.com/insomniacslk/dhcp/dhcpv4"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	kubevirtv1 "kubevirt.io/api/core/v1"

	"github.com/cldmnky/hyperdhcp/internal/dhcp/plugins/kubevirt/client/versioned/fake"
)

func TestSetupKubevirt(t *testing.T) {
	// Test case 1: Valid argument
	handler, err := setupKubevirt(clientcmd.RecommendedHomeFile)
	assert.NoError(t, err)
	assert.NotNil(t, handler)

	// Test case 2: Invalid argument
	handler, err = setupKubevirt()
	assert.Error(t, err)
	assert.Nil(t, handler)
}

func TestKubevirtHandler4(t *testing.T) {
	k := &KubevirtState{
		Client: fake.NewSimpleClientset(),
	}
	req := &dhcpv4.DHCPv4{
		ClientHWAddr: net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
	}
	resp := &dhcpv4.DHCPv4{}
	// add instancee to fake client
	k.Client.KubevirtV1().VirtualMachineInstances("test").Create(context.Background(), &kubevirtv1.VirtualMachineInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test",
		},
		Spec: kubevirtv1.VirtualMachineInstanceSpec{},
		Status: kubevirtv1.VirtualMachineInstanceStatus{
			Interfaces: []kubevirtv1.VirtualMachineInstanceNetworkInterface{
				{
					IP:  "10.202.2.2",
					MAC: "00:11:22:33:44:55",
				},
			},
		},
	}, metav1.CreateOptions{})
	expectedResp := resp
	expectedContinue := false
	actualResp, actualContinue := k.kubevirtHandler4(req, resp)
	assert.Equal(t, expectedResp, actualResp)
	assert.Equal(t, expectedContinue, actualContinue)
}
