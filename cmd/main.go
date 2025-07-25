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
	"strings"

	controllers "github.com/grafana/k6-operator/internal/controller"
	"github.com/grafana/k6-operator/pkg/plz"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

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
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&healthAddr, "health-probe-bind-address", ":8081", "The address the health endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")

	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

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
		HealthProbeBindAddress:     healthAddr,
	}

	if watchNamespaces, multiNamespaced := getWatchNamespaces(); multiNamespaced {
		defaultNamespaces := make(map[string]cache.Config, len(watchNamespaces))
		for _, ns := range watchNamespaces {
			defaultNamespaces[ns] = cache.Config{}
		}
		mgrOpts.Cache = cache.Options{
			DefaultNamespaces: defaultNamespaces,
		}
		setupLog.Info("WATCH_NAMESPACES is configured, WATCH_NAMESPACE will be ignored", "ns", watchNamespaces)
	} else if watchNamespace, namespaced := getWatchNamespace(); namespaced {
		mgrOpts.Cache = cache.Options{
			DefaultNamespaces: map[string]cache.Config{
				watchNamespace: {},
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

	if err = (&controllers.TestRunReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("TestRun"),
		Scheme: mgr.GetScheme(),
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

	plz.SetScheme(scheme)

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

func getWatchNamespaces() ([]string, bool) {
	const watchNamespacesEnvVar = "WATCH_NAMESPACES"

	if nsList, isSet := os.LookupEnv(watchNamespacesEnvVar); isSet {
		// The Kubernetes docs state that namespace names can only contain contain lowercase
		// alphanumeric characters or '-', making a comma (',') a valid separator for multiple namespaces.
		// See: https://kubernetes.io/docs/tasks/administer-cluster/namespaces/#creating-a-new-namespace
		// See: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#dns-label-names
		return strings.Split(nsList, ","), true
	}

	return nil, false
}
