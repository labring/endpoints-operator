/*
Copyright 2022 The sealos Authors.

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

package app

import (
	"context"
	"errors"
	"fmt"
	client2 "github.com/labring/endpoints-operator/utils/client"
	"os"
	"sort"

	"github.com/labring/endpoints-operator/apis/network/v1beta1"
	"github.com/labring/endpoints-operator/cmd/cepctl/app/options"
	"github.com/labring/operator-sdk/version"
	"github.com/spf13/cobra"
	v1 "k8s.io/api/core/v1"
	v1opts "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/json"
	cliflag "k8s.io/component-base/cli/flag"
	"k8s.io/component-base/term"
	"k8s.io/klog/v2"
	"sigs.k8s.io/yaml"
)

func NewCommand() *cobra.Command {
	s := options.NewOptions()

	cmd := &cobra.Command{
		Use:   "cepctl",
		Short: "cepctl is cli for cluster-endpoint",
		Run: func(cmd *cobra.Command, args []string) {
			if errs := s.Validate(); len(errs) != 0 {
				klog.Error(utilerrors.NewAggregate(errs))
				os.Exit(1)
			}
			if err := run(s, context.Background()); err != nil {
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
	if s.Version {
		if s.Short {
			fmt.Printf("Version: %s\n", version.Get().GitVersion)
		} else {
			fmt.Printf("Version: %s\n", fmt.Sprintf("%#v", version.Get()))
		}
		return nil
	}
	cli := client2.NewKubernetesClient(client2.NewKubernetesOptions(s.KubeConfig, s.Master))
	if cli == nil {
		return errors.New("build kube client error")
	}
	cep := &v1beta1.ClusterEndpoint{}
	cep.Namespace = s.Namespace
	cep.Name = s.Name
	cep.Spec.PeriodSeconds = s.PeriodSeconds
	svc, err := cli.Kubernetes().CoreV1().Services(s.Namespace).Get(ctx, s.Name, v1opts.GetOptions{})
	if err != nil {
		return err
	}
	klog.V(4).InfoS("get service", "name", s.Name, "namespace", s.Namespace, "spec", svc.Spec)
	if svc.Spec.ClusterIP == v1.ClusterIPNone {
		return errors.New("not support clusterIP=None service")
	}
	cep.Spec.ClusterIP = svc.Spec.ClusterIP
	ep, _ := cli.Kubernetes().CoreV1().Endpoints(s.Namespace).Get(ctx, s.Name, v1opts.GetOptions{})
	if ep != nil {
		klog.V(4).InfoS("get endpoint", "name", s.Name, "namespace", s.Namespace, "subsets", ep.Subsets)
		if len(ep.Subsets) > 1 {
			return errors.New("not support endpoint subsets length more than 1. Please spilt it")
		}
		ports := make([]v1beta1.ServicePort, 0)
		for _, subset := range ep.Subsets {
			enable := s.Probe

			ips := make([]string, 0)
			for _, addr := range subset.Addresses {
				ips = append(ips, addr.IP)
			}
			sort.Sort(sort.StringSlice(ips))
			for _, port := range subset.Ports {
				ports = append(ports, v1beta1.ServicePort{
					Handler: v1beta1.Handler{
						TCPSocket: &v1beta1.TCPSocketAction{Enable: enable},
					},
					TimeoutSeconds:   1,
					SuccessThreshold: 1,
					FailureThreshold: 3,
					Name:             port.Name,
					Protocol:         port.Protocol,
					Port:             findPortInSvc(svc, port.Name),
					TargetPort:       port.Port,
					Hosts:            ips,
				})
			}

		}
		cep.Spec.Ports = ports
	}
	configJson, _ := json.Marshal(cep)
	configYaml, _ := yaml.Marshal(cep)
	klog.V(4).InfoS("generator cep", "name", s.Name, "namespace", s.Namespace, "config", string(configJson))
	if s.Output != "" {
		if s.Output == "yaml" {
			println(string(configYaml))
			return nil
		}
		if s.Output == "json" {
			println(string(configJson))
			return nil
		}
	}
	c := client2.NewCep(cli.KubernetesDynamic())
	return c.CreateCR(ctx, cep)
}

func findPortInSvc(svc *v1.Service, portName string) int32 {

	if svc != nil {
		for _, p := range svc.Spec.Ports {
			if p.Name == portName {
				return p.Port
			}
		}
	}
	return 0
}
