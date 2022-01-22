// Copyright © 2022 cuisongliu@qq.com.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package app

import (
	"context"
	"fmt"
	"github.com/sealyun/endpoints-operator/cmd/endpoints-operator/app/options"
	"github.com/sealyun/endpoints-operator/controllers"
	"k8s.io/component-base/term"
	"net/http"
	"os"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/runtime"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	cliflag "k8s.io/component-base/cli/flag"
	"k8s.io/klog"
	"k8s.io/klog/klogr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

var (
	scheme = runtime.NewScheme()
)

func NewCommand() *cobra.Command {
	s := options.NewOptions()
	// make sure LeaderElection is not nil
	s = &options.Options{
		LeaderElection: s.LeaderElection,
		LeaderElect:    s.LeaderElect,
	}

	cmd := &cobra.Command{
		Use:  "endpoints-operator",
		Long: `endpoints-operator controller manager is a daemon that`,
		Run: func(cmd *cobra.Command, args []string) {
			if errs := s.Validate(); len(errs) != 0 {
				klog.Error(utilerrors.NewAggregate(errs))
				os.Exit(1)
			}
			if err := run(s, signals.SetupSignalHandler()); err != nil {
				klog.Error(err)
				os.Exit(1)
			}
		},
		SilenceUsage: true,
	}

	fs := cmd.Flags()
	namedFlagSets := s.Flags()

	for _, f := range namedFlagSets.FlagSets {
		fs.AddFlagSet(f)
	}

	usageFmt := "Usage:\n  %s\n"
	cols, _, _ := term.TerminalSize(cmd.OutOrStdout())
	cmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\n\n"+usageFmt, cmd.Long, cmd.UseLine())
		cliflag.PrintSections(cmd.OutOrStdout(), namedFlagSets, cols)
	})
	return cmd
}

func run(s *options.Options, ctx context.Context) error {
	mgrOptions := manager.Options{}
	if s.LeaderElect {
		mgrOptions = manager.Options{
			LeaderElection:             s.LeaderElect,
			LeaderElectionNamespace:    "kube-system",
			LeaderElectionID:           "sealyun-endpoints-operator-leader-election",
			LeaderElectionResourceLock: s.LeaderElectionResourceLock,
			LeaseDuration:              &s.LeaderElection.LeaseDuration,
			RetryPeriod:                &s.LeaderElection.RetryPeriod,
			RenewDeadline:              &s.LeaderElection.RenewDeadline,
		}
	}
	mgrOptions.Scheme = scheme
	mgrOptions.HealthProbeBindAddress = ":8080"
	mgrOptions.MetricsBindAddress = ":9090"
	klog.V(0).Info("setting up manager")
	ctrl.SetLogger(klogr.New())
	// Use 8443 instead of 443 cause we need root permission to bind port 443
	mgr, err := manager.New(ctrl.GetConfigOrDie(), mgrOptions)
	if err != nil {
		klog.Fatalf("unable to set up overall controller manager: %v", err)
	}
	klog.V(4).Info("[****] MaxConcurrent value is ", s.MaxConcurrent)
	klog.V(4).Info("[****] MaxRetry value is ", s.MaxRetry)

	controllers.Install(scheme)
	clusterReconciler := &controllers.Reconciler{}
	if s.MaxConcurrent > 0 {
		clusterReconciler.WorkNum = s.MaxConcurrent
	} else {
		clusterReconciler.WorkNum = 1
	}
	if s.MaxRetry > 0 {
		clusterReconciler.RetryCount = s.MaxRetry
	} else {
		clusterReconciler.RetryCount = 1
	}

	if err = clusterReconciler.SetupWithManager(mgr); err != nil {
		klog.Fatal("Unable to create cluster controller ", err)
	}

	klog.V(0).Info("Starting the controllers.")
	//healthz  Liveness
	if err := mgr.AddHealthzCheck("check", func(req *http.Request) error {
		return nil
	}); err != nil {
		klog.Fatal(err, "problem running manager liveness Check")
	}
	//readyz   Readiness
	if err := mgr.AddReadyzCheck("check", func(req *http.Request) error {
		return nil
	}); err != nil {
		klog.Fatal(err, "problem running manager readiness check")
	}

	if err = mgr.Start(ctx); err != nil {
		klog.Fatalf("unable to run the manager: %v", err)
	}
	return nil
}
