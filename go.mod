module github.com/sealyun/endpoints-operator

go 1.13

replace github.com/sealyun/endpoints-operator/library => ./library

require (
	github.com/go-logr/logr v0.1.0
	github.com/sealyun/endpoints-operator/library v0.0.0-00010101000000-000000000000
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
