/*


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
	"flag"
	"os"
	"time"

	"github.com/grafana/k6-operator/controllers"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"reconciler.io/runtime/reconcilers"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	"github.com/grafana/k6-operator/api/v1alpha1"
	k6v1alpha1 "github.com/grafana/k6-operator/api/v1alpha1"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(k6v1alpha1.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var healthAddr string
	var enableLeaderElection bool
	flag.StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&healthAddr, "health-addr", ":8081", "The address the health endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))

	mgrOpts := ctrl.Options{
		Scheme: scheme,
		Metrics: metricsserver.Options{
			BindAddress: metricsAddr,
		},
		WebhookServer: webhook.NewServer(webhook.Options{
			Port: 9443,
		}),
		LeaderElection:             enableLeaderElection,
		LeaderElectionID:           "fcdfce80.io",
		LeaderElectionResourceLock: "leases",
	}
	if watchNamespace, namespaced := getWatchNamespace(); namespaced {
		mgrOpts.Cache = cache.Options{
			DefaultNamespaces: map[string]cache.Config{
				watchNamespace: cache.Config{},
			},
		}
		setupLog.Info("WATCH_NAMESPACE is configured", "ns", watchNamespace)
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), mgrOpts)
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	_ = mgr.AddHealthzCheck("health", healthz.Ping)
	_ = mgr.AddReadyzCheck("ready", healthz.Ping)

	// duplicating the default value because there doesn't seem be any way to retrieve it...
	var cacheSyncPeriod = 10 * time.Hour
	if err = (controllers.NewK6Reconciler(&controllers.TestRunReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("K6"),
		Scheme: mgr.GetScheme(),
		Config: reconcilers.NewConfig(mgr, &v1alpha1.TestRun{}, cacheSyncPeriod),
	})).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "K6")
		os.Exit(1)
	}
	if err = (&controllers.TestRunReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("TestRun"),
		Scheme: mgr.GetScheme(),
		Config: reconcilers.NewConfig(mgr, &v1alpha1.TestRun{}, cacheSyncPeriod),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "TestRun")
		os.Exit(1)
	}
	if err = (&controllers.PrivateLoadZoneReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("PrivateLoadZone"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "PrivateLoadZone")
		os.Exit(1)
	}

	// +kubebuilder:scaffold:builder

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

func getWatchNamespace() (string, bool) {
	var watchNamespaceEnvVar = "WATCH_NAMESPACE"

	return os.LookupEnv(watchNamespaceEnvVar)
}
