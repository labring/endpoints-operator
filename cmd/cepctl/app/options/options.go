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
	PeriodSeconds int32
}

func NewOptions() *Options {
	s := &Options{
		Master:        "",
		KubeConfig:    path.Join(file.GetUserHomeDir(), ".kube", "config"),
		PeriodSeconds: 10,
		Name:          "",
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

	probe := fss.FlagSet("probe")
	probe.Int32Var(&s.PeriodSeconds, "periodSeconds", s.PeriodSeconds, "How often (in seconds) to perform the probe.Default is 10.")

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
	if s.PeriodSeconds < 0 {
		errs = append(errs, errors.New("param periodSeconds must more than zero"))
	}
	if !file.Exist(s.KubeConfig) {
		errs = append(errs, fmt.Errorf("kubeconfig path %s is not exist", s.KubeConfig))
	}
	if len(s.Name) == 0 {
		errs = append(errs, errors.New("service name must not empty"))
	}
	return errs
}
