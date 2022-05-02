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
	"github.com/labring/endpoints-operator/api/network/v1beta1"
	"github.com/labring/endpoints-operator/client"
	"github.com/labring/endpoints-operator/cmd/cepctl/app/options"
	"github.com/labring/endpoints-operator/library/version"
	"github.com/spf13/cobra"
	v1 "k8s.io/api/core/v1"
	v1opts "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/json"
	cliflag "k8s.io/component-base/cli/flag"
	"k8s.io/component-base/term"
	"k8s.io/klog/v2"
	"os"
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
	cli := client.NewKubernetesClient(client.NewKubernetesOptions(s.KubeConfig, s.Master))
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
	if svc.Spec.Selector != nil && len(svc.Spec.Selector) != 0 {
		return errors.New("not support selector not empty service")
	}
	ports := make([]v1beta1.ServicePort, len(svc.Spec.Ports))
	for i, p := range svc.Spec.Ports {
		enable := s.Probe
		ports[i] = v1beta1.ServicePort{
			Handler: v1beta1.Handler{
				TCPSocket: &v1beta1.TCPSocketAction{Enable: enable},
			},
			TimeoutSeconds:   1,
			SuccessThreshold: 1,
			FailureThreshold: 3,
			Name:             p.Name,
			Protocol:         p.Protocol,
			Port:             p.Port,
			TargetPort:       p.TargetPort.IntVal,
		}
	}
	cep.Spec.Ports = ports
	cep.Spec.ClusterIP = svc.Spec.ClusterIP
	ep, _ := cli.Kubernetes().CoreV1().Endpoints(s.Namespace).Get(ctx, s.Name, v1opts.GetOptions{})
	if ep != nil {
		klog.V(4).InfoS("get endpoint", "name", s.Name, "namespace", s.Namespace, "subsets", ep.Subsets)
		if len(ep.Subsets) > 1 {
			return errors.New("not support endpoint subsets length more than 1. Please spilt it")
		}
		cep.Spec.Hosts = convertAddress(ep.Subsets[0].Addresses)
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
	c := client.NewCep(cli.KubernetesDynamic())
	return c.CreateCR(ctx, cep)
}

func convertAddress(addresses []v1.EndpointAddress) []string {
	eas := make([]string, 0)
	for _, s := range addresses {
		eas = append(eas, s.IP)
	}
	return eas
}
