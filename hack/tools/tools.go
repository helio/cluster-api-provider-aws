//go:build tools
// +build tools

/*
Copyright 2019 The Kubernetes Authors.

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

// This package imports things required by build scripts, to force `go mod` to see them as dependencies
package tools

import (
	_ "embed"

	_ "github.com/a8m/envsubst"
	_ "github.com/ahmetb/gen-crd-api-reference-docs"
	_ "github.com/golang/mock/mockgen"
	_ "github.com/goreleaser/goreleaser"
	_ "github.com/itchyny/gojq/cmd/gojq"
	_ "github.com/joelanford/go-apidiff"
	_ "github.com/mikefarah/yq/v4"
	_ "k8s.io/apimachinery/pkg/util/intstr"
	_ "k8s.io/code-generator"
	_ "k8s.io/code-generator/cmd/conversion-gen"
	_ "sigs.k8s.io/cluster-api/hack/tools/conversion-verifier"
	_ "sigs.k8s.io/cluster-api/hack/tools/mdbook/embed"
	_ "sigs.k8s.io/cluster-api/hack/tools/mdbook/releaselink"
	_ "sigs.k8s.io/controller-runtime/tools/setup-envtest"
	_ "sigs.k8s.io/controller-tools/cmd/controller-gen"
	_ "sigs.k8s.io/kind"
	_ "sigs.k8s.io/kustomize/kustomize/v5"
	_ "sigs.k8s.io/promo-tools/v4/cmd/kpromo"
	_ "sigs.k8s.io/testing_frameworks/integration"
)
