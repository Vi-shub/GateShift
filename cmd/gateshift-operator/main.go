package main

import (
	"flag"
	"os"

	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	gateshiftv1alpha1 "github.com/gateshift/gateshift/api/v1alpha1"
	"github.com/gateshift/gateshift/internal/controller"
)

var scheme = runtime.NewScheme()

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(networkingv1.AddToScheme(scheme))
	utilruntime.Must(gateshiftv1alpha1.AddToScheme(scheme))
}

func main() {
	var metricsAddr string
	var probeAddr string
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "Prometheus metrics bind address")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "Health probe bind address")
	opts := zap.Options{Development: true}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
		Metrics: metricsserver.Options{
			BindAddress: metricsAddr,
		},
		HealthProbeBindAddress: probeAddr,
	})
	if err != nil {
		setupLog := ctrl.Log.WithName("setup")
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err := (&controller.MigrationRequestReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		ctrl.Log.WithName("setup").Error(err, "unable to create controller")
		os.Exit(1)
	}

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		ctrl.Log.WithName("setup").Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		ctrl.Log.WithName("setup").Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	ctrl.Log.Info("starting GateShift operator")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		ctrl.Log.WithName("setup").Error(err, "problem running manager")
		os.Exit(1)
	}
}
