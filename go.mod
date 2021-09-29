module github.com/ereslibre/kube-webhook-wrapper

go 1.15

require (
	github.com/go-logr/logr v0.4.0
	k8s.io/api v0.22.1
	k8s.io/apimachinery v0.22.1
	k8s.io/client-go v0.22.1
	sigs.k8s.io/controller-runtime v0.10.0
)

replace (
	k8s.io/api => k8s.io/api v0.20.11
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.20.11
	k8s.io/apimachinery => k8s.io/apimachinery v0.20.11
	k8s.io/apiserver => k8s.io/apiserver v0.20.11
	k8s.io/client-go => k8s.io/client-go v0.20.11
	k8s.io/code-generator => k8s.io/code-generator v0.20.11
	k8s.io/component-base => k8s.io/component-base v0.20.11
)
