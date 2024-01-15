package controller

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	serverv1beta1 "github.com/cldmnky/hyperdhcp/api/v1beta1"
)

var _ = Describe("Server controller", func() {
	const (
		serverName      = "test-server"
		serverNamespace = "default"

		timeout  = time.Second * 10
		duration = time.Second * 10
		interval = time.Millisecond * 250
	)

	Context("When creating a server", func() {
		It("Should create a server", func() {
			By("By creating a new server")
			ctx := context.Background()
			fiveParsed, err := time.ParseDuration("5m")
			Expect(err).NotTo(HaveOccurred())
			server := &serverv1beta1.Server{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "hyperdhcp.blahonga.me/v1beta1",
					Kind:       "Server",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      serverName,
					Namespace: serverNamespace,
				},
				Spec: serverv1beta1.ServerSpec{
					DHCPConfig: serverv1beta1.DHCPConfigSpec{
						DNS: []string{"192.168.1.1"},
						Range: serverv1beta1.DHCPRangeSpec{
							Start:     "10.202.2.10",
							End:       "10.202.2.20",
							LeaseTime: &metav1.Duration{Duration: fiveParsed},
						},
						Router:     "10.202.0.1",
						SubnetMask: "255.255.253.0",
					},
					NetworkAttachment: serverv1beta1.NetworkAttachmentSpec{
						Name:      "test-net",
						NameSpace: "default",
						IPs:       []string{"10.202.123.1"},
					},
				},
			}
			Expect(k8sClient.Create(ctx, server)).Should(Succeed())

			By("By checking that the server has been created")
			serverLookupKey := types.NamespacedName{Name: serverName, Namespace: serverNamespace}
			createdDeployment := &appsv1.Deployment{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, serverLookupKey, createdDeployment)
				return err == nil
			}, timeout, interval).Should(BeTrue())
			Expect(createdDeployment.Spec.Template.Spec.Containers[0].Image).To(Equal(DHCPImage))
		})
	})
})
