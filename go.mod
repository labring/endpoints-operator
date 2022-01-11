module github.com/cuisongliu/endpoints-balance

go 1.13

replace github.com/cuisongliu/endpoints-balance/library => ./library

require (
	github.com/cuisongliu/endpoints-balance/library v0.0.0-00010101000000-000000000000
	github.com/go-logr/logr v0.1.0
	github.com/spf13/cobra v0.0.5
	github.com/spf13/pflag v1.0.5
	k8s.io/api v0.17.2
	k8s.io/apimachinery v0.17.2
	k8s.io/apiserver v0.17.2
	k8s.io/client-go v0.17.2
	k8s.io/component-base v0.17.2
	k8s.io/klog v1.0.0
	sigs.k8s.io/controller-runtime v0.5.0
)
