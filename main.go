/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"flag"
	"os"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/isac322/static-lb/controllers"
	"github.com/isac322/static-lb/internal/application"
	"github.com/isac322/static-lb/internal/infrastructure"
	"github.com/isac322/static-lb/internal/presentation"
	//+kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	//+kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	var internalIPMappings presentation.IPMappingTargets
	var externalIPMappings presentation.IPMappingTargets
	var includeIngressIPFilter presentation.IPNetFilterFlag
	var includeExternalIPFilter presentation.IPNetFilterFlag
	var excludeIngressIPFilter presentation.IPNetFilterFlag
	var excludeExternalIPFilter presentation.IPNetFilterFlag

	flag.StringVar(
		&metricsAddr,
		"metrics-bind-address",
		":8080",
		"The address the metric endpoint binds to.",
	)
	flag.StringVar(
		&probeAddr,
		"health-probe-bind-address",
		":8081",
		"The address the probe endpoint binds to.",
	)
	flag.BoolVar(
		&enableLeaderElection,
		"leader-elect",
		false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.",
	)
	flag.Var(
		&internalIPMappings,
		"internal-ip-mapping",
		"where to assign node's internal ips (enum: ingress, external).",
	)
	flag.Var(
		&externalIPMappings,
		"external-ip-mapping",
		"where to assign node's external ips (enum: ingress, external).",
	)
	flag.Var(
		&includeIngressIPFilter,
		"include-ingress-ip-net",
		"IP networks that filters Ingress IP candidates before assign. (default: empty)",
	)
	flag.Var(
		&includeExternalIPFilter,
		"include-external-ip-net",
		"IP networks that filters External IP candidates before assign. (default: empty)",
	)
	flag.Var(
		&excludeIngressIPFilter,
		"exclude-ingress-ip-net",
		"IP networks that filters Ingress IP candidates out before assign. (default: empty)",
	)
	flag.Var(
		&excludeExternalIPFilter,
		"exclude-external-ip-net",
		"IP networks that filters External IP candidates out before assign. (default: empty)",
	)
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "899c85cb.bhyoo.com",
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

	var (
		nodeRepo          = infrastructure.NewNodeRepository(mgr.GetClient())
		svcRepo           = infrastructure.NewServiceRepository(mgr.GetClient())
		endpointSliceRepo = infrastructure.NewEndpointSliceRepository(mgr.GetClient())
		usecase           = application.New(
			endpointSliceRepo,
			nodeRepo,
			svcRepo,
			internalIPMappings.Mappings(),
			externalIPMappings.Mappings(),
			includeIngressIPFilter,
			includeExternalIPFilter,
			excludeIngressIPFilter,
			excludeExternalIPFilter,
		)
	)

	if err = endpointSliceRepo.RegisterFieldIndex(context.Background(), mgr.GetFieldIndexer()); err != nil {
		setupLog.Error(err, "unable to register index", "resource", "EndpointSlice")
		os.Exit(1)
	}

	if err = (&controllers.ServiceReconciler{
		Client:  mgr.GetClient(),
		Scheme:  mgr.GetScheme(),
		Usecase: usecase,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Service")
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
}
