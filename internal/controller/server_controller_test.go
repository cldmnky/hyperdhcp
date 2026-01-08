package controller

import (
	"context"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
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
						DNS:      []string{"192.168.1.1"},
						ServerID: "10.202.0.1",
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

		It("Should create a ConfigMap with correct configuration", func() {
			By("By checking that the ConfigMap has been created")
			ctx := context.Background()
			serverLookupKey := types.NamespacedName{Name: serverName, Namespace: serverNamespace}
			createdConfigMap := &corev1.ConfigMap{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, serverLookupKey, createdConfigMap)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			By("By verifying ConfigMap data")
			Expect(createdConfigMap.Data).To(HaveKey("hyperdhcp.yaml"))
			config := createdConfigMap.Data["hyperdhcp.yaml"]
			Expect(config).To(ContainSubstring("address: [\"10.202.123.1\"]"))
			Expect(config).To(ContainSubstring("range: 10.202.2.10-10.202.2.20"))
			Expect(config).To(ContainSubstring("router: 10.202.0.1"))
			Expect(config).To(ContainSubstring("dns: 192.168.1.1"))
			Expect(config).To(ContainSubstring("subnetMask: 255.255.253.0"))
			Expect(config).To(ContainSubstring("leaseTime: 5m0s"))
			Expect(config).To(ContainSubstring("kubevirt:"))
			Expect(config).To(ContainSubstring("enabled: true"))
		})

		It("Should create a PersistentVolumeClaim", func() {
			By("By checking that the PVC has been created")
			ctx := context.Background()
			serverLookupKey := types.NamespacedName{Name: serverName, Namespace: serverNamespace}
			createdPVC := &corev1.PersistentVolumeClaim{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, serverLookupKey, createdPVC)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			By("By verifying PVC specs")
			Expect(createdPVC.Spec.AccessModes).To(ContainElement(corev1.ReadWriteOnce))
			Expect(*createdPVC.Spec.StorageClassName).To(Equal("longhorn"))
			expectedStorage := resource.MustParse("25Mi")
			actualStorage := createdPVC.Spec.Resources.Requests[corev1.ResourceStorage]
			Expect(actualStorage.Equal(expectedStorage)).To(BeTrue())
		})

		It("Should create a ServiceAccount", func() {
			By("By checking that the ServiceAccount has been created")
			ctx := context.Background()
			serverLookupKey := types.NamespacedName{Name: serverName, Namespace: serverNamespace}
			createdSA := &corev1.ServiceAccount{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, serverLookupKey, createdSA)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			By("By verifying ServiceAccount labels")
			Expect(createdSA.Labels).To(HaveKeyWithValue("app", serverName))
		})

		It("Should create a Deployment with correct configuration", func() {
			By("By checking that the Deployment has been created")
			ctx := context.Background()
			serverLookupKey := types.NamespacedName{Name: serverName, Namespace: serverNamespace}
			createdDeployment := &appsv1.Deployment{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, serverLookupKey, createdDeployment)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			By("By verifying container args")
			container := createdDeployment.Spec.Template.Spec.Containers[0]
			Expect(container.Args).To(Equal([]string{"server", "--config", "/etc/dhcp/hyperdhcp.yaml"}))

			By("By verifying service account")
			Expect(createdDeployment.Spec.Template.Spec.ServiceAccountName).To(Equal(serverName))

			By("By verifying volume mounts")
			Expect(container.VolumeMounts).To(ContainElement(corev1.VolumeMount{
				Name:      "dhcp-config",
				MountPath: "/etc/dhcp",
				ReadOnly:  true,
			}))
			Expect(container.VolumeMounts).To(ContainElement(corev1.VolumeMount{
				Name:      "dhcp-leases",
				MountPath: "/var/lib/dhcp",
			}))

			By("By verifying volumes")
			volumes := createdDeployment.Spec.Template.Spec.Volumes

			// Check ConfigMap volume
			var foundConfigMapVolume bool
			for _, vol := range volumes {
				if vol.Name == "dhcp-config" && vol.VolumeSource.ConfigMap != nil {
					Expect(vol.VolumeSource.ConfigMap.LocalObjectReference.Name).To(Equal(serverName))
					Expect(vol.VolumeSource.ConfigMap.Items).To(HaveLen(1))
					Expect(vol.VolumeSource.ConfigMap.Items[0].Key).To(Equal("hyperdhcp.yaml"))
					Expect(vol.VolumeSource.ConfigMap.Items[0].Path).To(Equal("hyperdhcp.yaml"))
					foundConfigMapVolume = true
					break
				}
			}
			Expect(foundConfigMapVolume).To(BeTrue(), "ConfigMap volume not found")

			// Check PVC volume
			var foundPVCVolume bool
			for _, vol := range volumes {
				if vol.Name == "dhcp-leases" && vol.VolumeSource.PersistentVolumeClaim != nil {
					Expect(vol.VolumeSource.PersistentVolumeClaim.ClaimName).To(Equal(serverName))
					foundPVCVolume = true
					break
				}
			}
			Expect(foundPVCVolume).To(BeTrue(), "PVC volume not found")

			By("By verifying network attachment annotations")
			annotations := createdDeployment.Spec.Template.ObjectMeta.Annotations
			Expect(annotations).To(HaveKey("k8s.v1.cni.cncf.io/networks"))
			Expect(annotations["k8s.v1.cni.cncf.io/networks"]).To(ContainSubstring("test-net"))
			Expect(annotations["k8s.v1.cni.cncf.io/networks"]).To(ContainSubstring("10.202.123.1"))

			By("By verifying security context")
			Expect(container.SecurityContext.Privileged).NotTo(BeNil())
			Expect(*container.SecurityContext.Privileged).To(BeTrue())
			Expect(container.SecurityContext.RunAsUser).NotTo(BeNil())
			Expect(*container.SecurityContext.RunAsUser).To(Equal(int64(1000)))
		})
	})

	Context("When updating a server", func() {
		It("Should update ConfigMap when DNS is changed", func() {
			By("By creating a new server for update test")
			ctx := context.Background()
			updateServerName := "update-test-server"
			fiveParsed, err := time.ParseDuration("5m")
			Expect(err).NotTo(HaveOccurred())

			server := &serverv1beta1.Server{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "hyperdhcp.blahonga.me/v1beta1",
					Kind:       "Server",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      updateServerName,
					Namespace: serverNamespace,
				},
				Spec: serverv1beta1.ServerSpec{
					DHCPConfig: serverv1beta1.DHCPConfigSpec{
						DNS:      []string{"192.168.1.1"},
						ServerID: "10.202.0.1",
						Range: serverv1beta1.DHCPRangeSpec{
							Start:     "10.202.3.10",
							End:       "10.202.3.20",
							LeaseTime: &metav1.Duration{Duration: fiveParsed},
						},
						Router:     "10.202.0.1",
						SubnetMask: "255.255.253.0",
					},
					NetworkAttachment: serverv1beta1.NetworkAttachmentSpec{
						Name:      "test-net",
						NameSpace: "default",
						IPs:       []string{"10.202.124.1"},
					},
				},
			}
			Expect(k8sClient.Create(ctx, server)).Should(Succeed())

			By("By waiting for ConfigMap to be created")
			serverLookupKey := types.NamespacedName{Name: updateServerName, Namespace: serverNamespace}
			createdConfigMap := &corev1.ConfigMap{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, serverLookupKey, createdConfigMap)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			// Verify initial DNS configuration
			Expect(createdConfigMap.Data["hyperdhcp.yaml"]).To(ContainSubstring("dns: 192.168.1.1"))

			By("By updating the server DNS configuration")
			Eventually(func() error {
				updatedServer := &serverv1beta1.Server{}
				if err := k8sClient.Get(ctx, serverLookupKey, updatedServer); err != nil {
					return err
				}
				updatedServer.Spec.DHCPConfig.DNS = []string{"8.8.8.8", "8.8.4.4"}
				return k8sClient.Update(ctx, updatedServer)
			}, timeout, interval).Should(Succeed())

			By("By verifying ConfigMap is updated after reconciliation")
			Eventually(func() bool {
				cm := &corev1.ConfigMap{}
				if err := k8sClient.Get(ctx, serverLookupKey, cm); err != nil {
					return false
				}
				config := cm.Data["hyperdhcp.yaml"]
				return strings.Contains(config, "dns: 8.8.8.8,8.8.4.4")
			}, timeout*2, interval).Should(BeTrue())

			By("By cleaning up the update test server")
			Expect(k8sClient.Delete(ctx, server)).Should(Succeed())
		})

		It("Should update ConfigMap when IP range is changed", func() {
			By("By creating a new server for range update test")
			ctx := context.Background()
			rangeServerName := "range-test-server"
			fiveParsed, err := time.ParseDuration("5m")
			Expect(err).NotTo(HaveOccurred())

			server := &serverv1beta1.Server{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "hyperdhcp.blahonga.me/v1beta1",
					Kind:       "Server",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      rangeServerName,
					Namespace: serverNamespace,
				},
				Spec: serverv1beta1.ServerSpec{
					DHCPConfig: serverv1beta1.DHCPConfigSpec{
						DNS:      []string{"192.168.1.1"},
						ServerID: "10.202.0.1",
						Range: serverv1beta1.DHCPRangeSpec{
							Start:     "10.202.4.10",
							End:       "10.202.4.20",
							LeaseTime: &metav1.Duration{Duration: fiveParsed},
						},
						Router:     "10.202.0.1",
						SubnetMask: "255.255.253.0",
					},
					NetworkAttachment: serverv1beta1.NetworkAttachmentSpec{
						Name:      "test-net",
						NameSpace: "default",
						IPs:       []string{"10.202.125.1"},
					},
				},
			}
			Expect(k8sClient.Create(ctx, server)).Should(Succeed())

			By("By waiting for ConfigMap to be created")
			serverLookupKey := types.NamespacedName{Name: rangeServerName, Namespace: serverNamespace}
			createdConfigMap := &corev1.ConfigMap{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, serverLookupKey, createdConfigMap)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			// Verify initial range configuration
			Expect(createdConfigMap.Data["hyperdhcp.yaml"]).To(ContainSubstring("range: 10.202.4.10-10.202.4.20"))

			By("By updating the server range configuration")
			Eventually(func() error {
				updatedServer := &serverv1beta1.Server{}
				if err := k8sClient.Get(ctx, serverLookupKey, updatedServer); err != nil {
					return err
				}
				updatedServer.Spec.DHCPConfig.Range.Start = "10.202.4.100"
				updatedServer.Spec.DHCPConfig.Range.End = "10.202.4.200"
				return k8sClient.Update(ctx, updatedServer)
			}, timeout, interval).Should(Succeed())

			By("By verifying ConfigMap is updated after reconciliation")
			Eventually(func() bool {
				cm := &corev1.ConfigMap{}
				if err := k8sClient.Get(ctx, serverLookupKey, cm); err != nil {
					return false
				}
				config := cm.Data["hyperdhcp.yaml"]
				return strings.Contains(config, "range: 10.202.4.100-10.202.4.200")
			}, timeout*2, interval).Should(BeTrue())

			By("By cleaning up the range test server")
			Expect(k8sClient.Delete(ctx, server)).Should(Succeed())
		})
	})

	Context("When deleting a server", func() {
		It("Should clean up the original test server", func() {
			By("By deleting the original test server")
			ctx := context.Background()
			serverLookupKey := types.NamespacedName{Name: serverName, Namespace: serverNamespace}

			server := &serverv1beta1.Server{}
			err := k8sClient.Get(ctx, serverLookupKey, server)
			if err == nil {
				Expect(k8sClient.Delete(ctx, server)).Should(Succeed())
			}
		})
	})
})
