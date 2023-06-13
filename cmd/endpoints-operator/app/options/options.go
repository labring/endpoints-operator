// Copyright © 2022 The sealos Authors.
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

package options

import (
	"flag"
	"strings"
	"time"

	"k8s.io/client-go/tools/leaderelection/resourcelock"

	"github.com/spf13/pflag"
	"k8s.io/client-go/tools/leaderelection"
	cliflag "k8s.io/component-base/cli/flag"
	"k8s.io/klog/v2"
)

type Options struct {
	LeaderElect                bool
	LeaderElection             *leaderelection.LeaderElectionConfig
	LeaderElectionResourceLock string
	MaxConcurrent              int
	MaxRetry                   int
	RateLimiterOptions         RateLimiterOptions
}

func NewOptions() *Options {
	s := &Options{
		LeaderElection: &leaderelection.LeaderElectionConfig{
			LeaseDuration: 15 * time.Second,
			RenewDeadline: 10 * time.Second,
			RetryPeriod:   2 * time.Second,
		},
		LeaderElect: false,
	}

	return s
}

func (s *Options) Flags() cliflag.NamedFlagSets {
	fss := cliflag.NamedFlagSets{}

	fs := fss.FlagSet("leaderelection")
	s.bindLeaderElectionFlags(s.LeaderElection, fs)

	fs.BoolVar(&s.LeaderElect, "leader-elect", s.LeaderElect, ""+
		"Whether to enable leader election. This field should be enabled when controller manager"+
		"deployed with multiple replicas.")

	kfs := fss.FlagSet("klog")
	local := flag.NewFlagSet("klog", flag.ExitOnError)
	klog.InitFlags(local)
	local.VisitAll(func(fl *flag.Flag) {
		fl.Name = strings.Replace(fl.Name, "_", "-", -1)
		kfs.AddGoFlag(fl)
	})

	// add MaxConcurrent args
	// MaxConcurrent this is the maximum number of concurrent Reconciles which can be run. Defaults to 1.
	mc := fss.FlagSet("worker")
	mc.IntVar(&s.MaxConcurrent, "maxconcurrent", 1, "MaxConcurrent this is the maximum number of concurrent Reconciles "+
		"which can be run. Defaults to 1.")
	mc.IntVar(&s.MaxRetry, "maxretry", 1, "MaxRetry this is the maximum number of retry liveliness "+
		"which can be run. Defaults to 1.")
	s.RateLimiterOptions.BindFlags(flag.CommandLine)
	return fss
}

func (s *Options) Validate() []error {
	var errs []error
	return errs
}

func (s *Options) bindLeaderElectionFlags(l *leaderelection.LeaderElectionConfig, fs *pflag.FlagSet) {
	fs.DurationVar(&l.LeaseDuration, "leader-elect-lease-duration", l.LeaseDuration, ""+
		"The duration that non-leader candidates will wait after observing a leadership "+
		"renewal until attempting to acquire leadership of a led but unrenewed leader "+
		"slot. This is effectively the maximum duration that a leader can be stopped "+
		"before it is replaced by another candidate. This is only applicable if leader "+
		"election is enabled.")
	fs.DurationVar(&l.RenewDeadline, "leader-elect-renew-deadline", l.RenewDeadline, ""+
		"The interval between attempts by the acting master to renew a leadership slot "+
		"before it stops leading. This must be less than or equal to the lease duration. "+
		"This is only applicable if leader election is enabled.")
	fs.DurationVar(&l.RetryPeriod, "leader-elect-retry-period", l.RetryPeriod, ""+
		"The duration the clients should wait between attempting acquisition and renewal "+
		"of a leadership. This is only applicable if leader election is enabled.")
	fs.StringVar(&s.LeaderElectionResourceLock, "leader-elect-resource-lock", resourcelock.ConfigMapsLeasesResourceLock,
		"Leader election resource lock, support: endpoints,configmaps,leases,endpointsleases,configmapsleases")

}
