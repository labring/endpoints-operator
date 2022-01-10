/*
Copyright 2022 cuisongliu@qq.com.

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
	"github.com/cuisongliu/endpoints-balance/library/version"
	"github.com/emicklei/go-restful"
	"github.com/go-logr/logr"
	"k8s.io/klog"
	"k8s.io/klog/klogr"
	"net/http"
	"os"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/cuisongliu/endpoints-balance/controllers"
	cliflag "k8s.io/component-base/cli/flag"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)

	_ = corev1.AddToScheme(scheme)
	// +kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr, healthAddr, versionAddr string
	var enableLeaderElection bool
	//var syncPeriod int64
	flag.StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&healthAddr, "health-addr", ":9090", "Health address. Readiness url is  /readyz, Liveness url is /healthz")
	flag.StringVar(&versionAddr, "version-addr", ":7070", "The address the version endpoint binds to. /version")
	flag.BoolVar(&enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")

	fss := cliflag.NamedFlagSets{}
	kfs := fss.FlagSet("klog")
	local := flag.NewFlagSet("klog", flag.ExitOnError)
	klog.InitFlags(local)
	local.VisitAll(func(fl *flag.Flag) {
		fl.Name = strings.Replace(fl.Name, "_", "-", -1)
		kfs.AddGoFlag(fl)
	})
	flag.Parse()
	ctrl.SetLogger(klogr.New())

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "1c09d714.sealyun.com",
		MetricsBindAddress:     metricsAddr,
		HealthProbeBindAddress: healthAddr,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err = (&controllers.ServiceReconciler{}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create service controller", "controller", "Service")
		os.Exit(1)
	}
	// +kubebuilder:scaffold:builder

	//healthz  Liveness
	if err := mgr.AddHealthzCheck("check", func(req *http.Request) error {
		return nil
	}); err != nil {
		setupLog.Error(err, "problem running manager liveness Check")
		os.Exit(1)
	}
	//readyz   Readiness
	if err := mgr.AddReadyzCheck("check", func(req *http.Request) error {
		return nil
	}); err != nil {
		setupLog.Error(err, "problem running manager readiness check")
		os.Exit(1)
	}
	go versionRegistry(versionAddr, setupLog)
	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

func versionRegistry(sPort string, logger logr.Logger) {
	wsContainer := restful.NewContainer()
	wsContainer.Router(restful.CurlyRouter{})
	scheduler := new(restful.WebService)
	scheduler.Path("").Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON) // you can specify this per route as well
	scheduler.Route(scheduler.GET("/version").To(func(request *restful.Request, response *restful.Response) {
		_ = response.WriteEntity(version.Get())
	}))
	wsContainer.Add(scheduler)
	server := &http.Server{Addr: sPort, Handler: wsContainer}
	if err := server.ListenAndServe(); err != nil {
		logger.Error(err, "problem running application")
	}
}
