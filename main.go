package main

import (
	"fmt"
	"os"

	"github.com/pulumi/pulumi-command/sdk/go/command/local"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"

	v1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	metaV1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {

		subcriptionId := "af4f0732-dd8c-4330-a357-b9593255c3f0"
		kubeconfigPath := os.Getenv("KUBECONFIGPATH")
		resourceGroupName := "grow-dev-cesare"
		myNamespace := "corda"

		//az login
		loginIntoAzCliResourceName := "loginIntoAzCli"
		existingResource, err := local.GetCommand(ctx, loginIntoAzCliResourceName, nil, nil)
		if err != nil || existingResource == nil {
			// If the resource doesn't exist, proceed to create it
			cmd, err := local.NewCommand(ctx, loginIntoAzCliResourceName, &local.CommandArgs{
				Create: pulumi.String("az account set --subscription " + subcriptionId),
			})
			if err != nil {
				return fmt.Errorf("error with az login: %v", err)
			}
			getKubeConfig, err := local.NewCommand(ctx, "local-aks-kube-config", &local.CommandArgs{
				Create: pulumi.String("az aks get-credentials --resource-group " + resourceGroupName + " --name " + resourceGroupName + " --overwrite-existing"),
			})
			if err != nil {
				return fmt.Errorf("error with kube config retrieval: %v", err)
			}
			ctx.Export("commandOutput", cmd.Stdout)
			ctx.Export("getKubeConfig", getKubeConfig.Stdout)

		} else {
			ctx.Log.Warn(fmt.Sprintf("Resource with name %s already exists", loginIntoAzCliResourceName), nil)
		}

		//Retrieves the Kubeconfig for the above AKS Cluster
		kubeconfig, err := os.ReadFile(kubeconfigPath)
		if err != nil {
			return fmt.Errorf("error whilst retrieving the local kubeconfig: %v", err)
		}
		k8sProvider, err := kubernetes.NewProvider(ctx, "aksK8sProvider", &kubernetes.ProviderArgs{
			Kubeconfig: pulumi.String(kubeconfig),
		})
		if err != nil {
			return fmt.Errorf("error whilst getting an existing kubeconfig file: %v", err)
		}
		ctx.Export("aksK8sProvider", k8sProvider.ID())

		//Create Corda Namespace
		cordaNamespace, err := v1.NewNamespace(ctx, "cordaNamespace", &v1.NamespaceArgs{
			Metadata: &metaV1.ObjectMetaArgs{
				Name: pulumi.String(myNamespace),
			},
		}, pulumi.Provider(k8sProvider))
		if err != nil {
			return err
		}
		ctx.Export("cordaNamespace", cordaNamespace.ID())

		// Initialize the Kubernetes Provider
		provider := NewProvider(ctx, k8sProvider, myNamespace)

		////////////////////////////
		// CORDA NODE /////////////
		///////////////////////////
		//Sucks into the Corda Network Kubernetes  configmap for the node
		nodeconfConfigMap, err := provider.CreateNodeconfConfigMap("node-configmap", "resources/provider/node.conf")
		if err != nil {
			return fmt.Errorf("error  with the creation of the corda node configmap: %v", err)
		}
		ctx.Export("nodeConfigMap", nodeconfConfigMap.URN())

		// Create Node Kubernetes PersistentVolumeClaims
		nodePvcNames := []string{"node-certificates-pvc", "node-config-pvc", "node-persistence-pvc", "node-logs-pvc", "node-configmap-pvc", "node-networkcertificate-configmap-pvc"}
		for _, pvcName := range nodePvcNames {
			_, err := provider.CreatePVC(pvcName)
			if err != nil {
				return err
			}
		}

		// Create the Corda Node (Provider) Deployment
		GetNetworkcertificate()
		networkcertificateConfigMap, err := provider.CreateNetworkcertificateConfigMap()
		if err != nil {
			return fmt.Errorf("error with the creation of the network certificate to connect to an existing CENM instace: %v", err)
		}
		nodeDeployment, err := provider.CreateDeployment("node", nodeconfConfigMap, networkcertificateConfigMap, nodePvcNames)
		if err != nil {
			return fmt.Errorf("error  with the creation of the corda node deployment: %v", err)
		}
		ctx.Export("nodeDeployment", nodeDeployment.ID())

		return nil
	})
}
