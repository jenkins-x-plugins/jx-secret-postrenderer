module github.com/jenkins-x-plugins/jx-secret-postrenderer

go 1.15

require (
	github.com/jenkins-x-plugins/jx-secret v0.1.29
	github.com/jenkins-x/go-scm v1.8.2
	github.com/jenkins-x/jx-helpers/v3 v3.0.114
	github.com/pkg/errors v0.9.1
	github.com/sethvargo/go-envconfig v0.3.2
	github.com/spf13/cobra v1.1.1
	github.com/stretchr/testify v1.7.0
	sigs.k8s.io/kustomize/kyaml v0.10.6
	sigs.k8s.io/yaml v1.2.0
)

replace (
	k8s.io/api => k8s.io/api v0.20.2
	k8s.io/apimachinery => k8s.io/apimachinery v0.20.2
	k8s.io/client-go => k8s.io/client-go v0.20.2
)
