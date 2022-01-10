module github.com/cuisongliu/endpoints-balance

go 1.13

replace github.com/cuisongliu/endpoints-balance/library => ./library

require (
	github.com/cuisongliu/endpoints-balance/library v0.0.0-00010101000000-000000000000
	github.com/emicklei/go-restful v2.9.5+incompatible
	github.com/go-logr/logr v0.1.0
	go.uber.org/zap v1.10.0
	k8s.io/api v0.17.2
	k8s.io/apimachinery v0.17.2
	k8s.io/client-go v0.17.2
	sigs.k8s.io/controller-runtime v0.5.0
)
