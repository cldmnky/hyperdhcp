package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"

	hyperdhcpv1beta1 "github.com/cldmnky/hyperdhcp/api/v1beta1"
	"github.com/cldmnky/hyperdhcp/internal/controller"
)

var (
	scheme               = runtime.NewScheme()
	setupLog             = ctrl.Log.WithName("setup")
	metricsAddr          string
	probeAddr            string
	enableLeaderElection bool
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(hyperdhcpv1beta1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
	// Add flags to the manager command
	// metrics-bind-address is the address the metric endpoint binds to
	managerCmd.Flags().StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	// health-probe-bind-address is the address the probe endpoint binds to
	managerCmd.Flags().StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	// leader-elect enables leader election for controller manager
	managerCmd.Flags().BoolVar(&enableLeaderElection, "leader-elect", false, "Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager.")
	// related images
	managerCmd.Flags().StringVar(&controller.DHCPImage, "dhcp-image", "quay.io/cldmnky/hyperdhcpd:latest", "The DHCP server image to use")
}

var managerCmd = &cobra.Command{
	Use:  "manager",
	Long: `Starts the DHCP operator manager`,
	Run: func(cmd *cobra.Command, args []string) {
		mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
			Scheme:                 scheme,
			MetricsBindAddress:     metricsAddr,
			Port:                   9443,
			HealthProbeBindAddress: probeAddr,
			LeaderElection:         enableLeaderElection,
			LeaderElectionID:       "2e89b68a.blahonga.me",
			// LeaderElectionReleaseOnCancel defines if the leader should step down voluntarily
			// when the Manager ends. This requires the binary to immediately end when the
			// Manager is stopped, otherwise, this setting is unsafe. Setting this significantly
			// speeds up voluntary leader transitions as the new leader don't have to wait
			// LeaseDuration time first.
			//
			// In the default scaffold provided, the program ends immediately after
			// the manager stops, so would be fine to enable this option. However,
			// if you are doing or is intended to do any operation such as perform cleanups
			// after the manager stops then its usage might be unsafe.
			// LeaderElectionReleaseOnCancel: true,
		})
		if err != nil {
			setupLog.Error(err, "unable to start manager")
			os.Exit(1)
		}
		setupLog.Info("viper config:", "dhcp-image", viper.GetString("related-image-dhcp"))
		if err = (&controller.ServerReconciler{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
		}).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "Server")
			os.Exit(1)
		}
		//+kubebuilder:scaffold:builder

		if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
			setupLog.Error(err, "unable to set up health check")
			os.Exit(1)
		}
		if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
			setupLog.Error(err, "unable to set up ready check")
			os.Exit(1)
		}

		setupLog.Info("starting manager")
		if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
			setupLog.Error(err, "problem running manager")
			os.Exit(1)
		}
	},
}
