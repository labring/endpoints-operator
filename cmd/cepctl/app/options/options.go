// Copyright © 2022 The sealyun Authors.
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
	"errors"
	"flag"
	"fmt"
	"github.com/sealyun/endpoints-operator/library/file"
	cliflag "k8s.io/component-base/cli/flag"
	"k8s.io/klog/v2"
	"path"
	"strings"
)

type Options struct {
	// Master is used to override the kubeconfig's URL to the apiserver.
	Master string
	//KubeConfig is the path to a KubeConfig file.
	KubeConfig    string
	Name          string
	Namespace     string
	PeriodSeconds int32
	Probe         bool
	Output        string
	Version       bool
	Short         bool
}

func NewOptions() *Options {
	s := &Options{
		Master:        "",
		KubeConfig:    path.Join(file.GetUserHomeDir(), ".kube", "config"),
		PeriodSeconds: 10,
		Name:          "",
		Probe:         false,
		Namespace:     "default",
		Output:        "",
		Version:       false,
		Short:         false,
	}
	return s
}

func (s *Options) Flags() cliflag.NamedFlagSets {
	fss := cliflag.NamedFlagSets{}

	kube := fss.FlagSet("kube")
	kube.StringVar(&s.KubeConfig, "kubeconfig", s.KubeConfig, "Path to kubeconfig file with authorization information (the master location can be overridden by the master flag).")
	kube.StringVar(&s.Master, "master", s.Master, "The address of the Kubernetes API server (overrides any value in kubeconfig).")

	cep := fss.FlagSet("cep")
	cep.StringVar(&s.Name, "service-name", s.Name, "Sync cap from service name.")
	cep.StringVar(&s.Namespace, "service-namespace", s.Namespace, "Sync cap from service namespace.")
	cep.StringVarP(&s.Output, "output", "o", s.Output, "output json|yaml. if not set,will create cep to kubernetes")

	probe := fss.FlagSet("probe")
	probe.Int32Var(&s.PeriodSeconds, "periodSeconds", s.PeriodSeconds, "How often (in seconds) to perform the probe.Default is 10.")
	probe.BoolVar(&s.Probe, "probe", s.Probe, "When set value is true,add default probe of tcpAction.")

	version := fss.FlagSet("version")
	version.BoolVar(&s.Version, "version", s.Version, "Print the client  version information")
	version.BoolVar(&s.Short, "short", s.Short, "If true, print just the version number.")

	kfs := fss.FlagSet("klog")
	local := flag.NewFlagSet("klog", flag.ExitOnError)
	klog.InitFlags(local)
	local.VisitAll(func(fl *flag.Flag) {
		fl.Name = strings.Replace(fl.Name, "_", "-", -1)
		kfs.AddGoFlag(fl)
	})

	return fss
}

func (s *Options) Validate() []error {
	var errs []error
	if s.Version {
		return errs
	}
	if s.PeriodSeconds < 0 {
		errs = append(errs, errors.New("param periodSeconds must more than zero"))
	}
	if !file.Exist(s.KubeConfig) {
		errs = append(errs, fmt.Errorf("kubeconfig path %s is not exist", s.KubeConfig))
	}
	if len(s.Name) == 0 {
		errs = append(errs, errors.New("service name must not empty"))
	}
	if len(s.Namespace) == 0 {
		errs = append(errs, errors.New("service namespace must not empty"))
	}
	if len(s.Output) != 0 {
		if s.Output != "yaml" && s.Output != "json" {
			errs = append(errs, errors.New("output must be is yaml or json"))
		}
	}
	return errs
}
