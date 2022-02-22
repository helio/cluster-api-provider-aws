//go:build e2e
// +build e2e

/*
Copyright 2020 The Kubernetes Authors.

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

package unmanaged

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/pointer"

	"sigs.k8s.io/cluster-api-provider-aws/test/e2e/shared"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/test/e2e"
	"sigs.k8s.io/cluster-api/test/framework"
	"sigs.k8s.io/cluster-api/test/framework/clusterctl"
	"sigs.k8s.io/cluster-api/util"
)

// MachinePoolInput is the input for MachinePoolSpec.
type MachinePoolInput struct {
	E2ECtx                *shared.E2EContext
	ClusterctlConfigPath  string
	BootstrapClusterProxy framework.ClusterProxy
	ArtifactFolder        string
	SkipCleanup           bool

	// Flavor, if specified must refer to a template that contains a MachinePool resource.
	// If not specified, "machine-pool" is used
	Flavor *string
}

// MachinePoolSpec implements a test that verifies MachinePool create, scale up and scale down.
func MachinePoolSpec(ctx context.Context, inputGetter func() MachinePoolInput) {
	var (
		specName         = "machine-pool"
		input            MachinePoolInput
		namespace        *corev1.Namespace
		cancelWatches    context.CancelFunc
		clusterResources *clusterctl.ApplyClusterTemplateAndWaitResult
	)

	BeforeEach(func() {
		Expect(ctx).NotTo(BeNil(), "ctx is required for %s spec", specName)
		input = inputGetter()
		Expect(input.E2ECtx.E2EConfig).ToNot(BeNil(), "Invalid argument. input.E2EConfig can't be nil when calling %s spec", specName)
		Expect(input.ClusterctlConfigPath).To(BeAnExistingFile(), "Invalid argument. input.ClusterctlConfigPath must be an existing file when calling %s spec", specName)
		Expect(input.BootstrapClusterProxy).ToNot(BeNil(), "Invalid argument. input.BootstrapClusterProxy can't be nil when calling %s spec", specName)
		Expect(os.MkdirAll(input.ArtifactFolder, 0755)).To(Succeed(), "Invalid argument. input.ArtifactFolder can't be created for %s spec", specName)
		Expect(input.E2ECtx.E2EConfig.Variables).To(HaveKey(shared.KubernetesVersion))
		Expect(input.E2ECtx.E2EConfig.Variables).To(e2e.HaveValidVersion(input.E2ECtx.E2EConfig.GetVariable(shared.KubernetesVersion)))

		// Setup a Namespace where to host objects for this spec and create a watcher for the namespace events.
		namespace = shared.SetupSpecNamespace(ctx, specName, input.E2ECtx)
		clusterResources = new(clusterctl.ApplyClusterTemplateAndWaitResult)
	})

	It("Should successfully create a cluster with machine pool machines", func() {
		By("Creating a workload cluster")
		workerMachineCount := int32(1)
		clusterctl.ApplyClusterTemplateAndWait(ctx, clusterctl.ApplyClusterTemplateAndWaitInput{
			ClusterProxy: input.BootstrapClusterProxy,
			ConfigCluster: clusterctl.ConfigClusterInput{
				LogFolder:                filepath.Join(input.ArtifactFolder, "clusters", input.BootstrapClusterProxy.GetName()),
				ClusterctlConfigPath:     input.ClusterctlConfigPath,
				KubeconfigPath:           input.BootstrapClusterProxy.GetKubeconfigPath(),
				InfrastructureProvider:   clusterctl.DefaultInfrastructureProvider,
				Flavor:                   pointer.StringDeref(input.Flavor, "machine-pool"),
				Namespace:                namespace.Name,
				ClusterName:              fmt.Sprintf("%s-%s", specName, util.RandomString(6)),
				KubernetesVersion:        input.E2ECtx.E2EConfig.GetVariable(shared.KubernetesVersion),
				ControlPlaneMachineCount: pointer.Int64Ptr(1),
				WorkerMachineCount:       pointer.Int64Ptr(int64(workerMachineCount)),
			},
			WaitForClusterIntervals:      input.E2ECtx.E2EConfig.GetIntervals(specName, "wait-cluster"),
			WaitForControlPlaneIntervals: input.E2ECtx.E2EConfig.GetIntervals(specName, "wait-control-plane"),
			WaitForMachinePools:          input.E2ECtx.E2EConfig.GetIntervals(specName, "wait-machine-pool-nodes"),
		}, clusterResources)

		By("Scaling the machine pool up")
		framework.ScaleMachinePoolAndWait(ctx, framework.ScaleMachinePoolAndWaitInput{
			ClusterProxy:              input.BootstrapClusterProxy,
			Cluster:                   clusterResources.Cluster,
			Replicas:                  workerMachineCount + 1,
			MachinePools:              clusterResources.MachinePools,
			WaitForMachinePoolToScale: input.E2ECtx.E2EConfig.GetIntervals(specName, "wait-machine-pool-nodes"),
		})

		By("Scaling the machine pool down to zero")
		framework.ScaleMachinePoolAndWait(ctx, framework.ScaleMachinePoolAndWaitInput{
			ClusterProxy:              input.BootstrapClusterProxy,
			Cluster:                   clusterResources.Cluster,
			Replicas:                  0,
			MachinePools:              clusterResources.MachinePools,
			WaitForMachinePoolToScale: input.E2ECtx.E2EConfig.GetIntervals(specName, "wait-machine-pool-nodes"),
		})

		By("PASSED!")
	})

	AfterEach(func() {
		// Dumps all the resources in the spec namespace, then cleanups the cluster object and the spec namespace itself.
		dumpSpecResourcesAndCleanup(ctx, specName, input.BootstrapClusterProxy, input.ArtifactFolder, namespace, cancelWatches, clusterResources.Cluster, input.E2ECtx.E2EConfig.GetIntervals, input.SkipCleanup)
	})
}

func dumpSpecResourcesAndCleanup(ctx context.Context, specName string, clusterProxy framework.ClusterProxy, artifactFolder string, namespace *corev1.Namespace, cancelWatches context.CancelFunc, cluster *clusterv1.Cluster, intervalsGetter func(spec, key string) []interface{}, skipCleanup bool) {
	shared.Byf("Dumping logs from the %q workload cluster", cluster.Name)

	// Dump all the logs from the workload cluster before deleting them.
	clusterProxy.CollectWorkloadClusterLogs(ctx, cluster.Namespace, cluster.Name, filepath.Join(artifactFolder, "clusters", cluster.Name))

	shared.Byf("Dumping all the Cluster API resources in the %q namespace", namespace.Name)

	// Dump all Cluster API related resources to artifacts before deleting them.
	framework.DumpAllResources(ctx, framework.DumpAllResourcesInput{
		Lister:    clusterProxy.GetClient(),
		Namespace: namespace.Name,
		LogPath:   filepath.Join(artifactFolder, "clusters", clusterProxy.GetName(), "resources"),
	})

	if !skipCleanup {
		shared.Byf("Deleting cluster %s/%s", cluster.Namespace, cluster.Name)
		// While https://github.com/kubernetes-sigs/cluster-api/issues/2955 is addressed in future iterations, there is a chance
		// that cluster variable is not set even if the cluster exists, so we are calling DeleteAllClustersAndWait
		// instead of DeleteClusterAndWait
		framework.DeleteAllClustersAndWait(ctx, framework.DeleteAllClustersAndWaitInput{
			Client:    clusterProxy.GetClient(),
			Namespace: namespace.Name,
		}, intervalsGetter(specName, "wait-delete-cluster")...)

		shared.Byf("Deleting namespace used for hosting the %q test spec", specName)
		framework.DeleteNamespace(ctx, framework.DeleteNamespaceInput{
			Deleter: clusterProxy.GetClient(),
			Name:    namespace.Name,
		})
	}
	cancelWatches()
}
