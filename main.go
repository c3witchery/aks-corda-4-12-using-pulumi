package main

import (
	"fmt"
	"os"

	"github.com/pulumi/pulumi-command/sdk/go/command/local"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {

		subcriptionId := "af4f0732-dd8c-4330-a357-b9593255c3f0"
		kubeconfigPath := os.Getenv("KUBECONFIGPATH")
		resourceGroupName := "grow-dev-cesare"

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

		kubeconfig, err := os.ReadFile(kubeconfigPath)
		if err != nil {
			return fmt.Errorf("error whilst retrieving the local kubeconfig: %v", err)
		}
		//Retrieves the Kubeconfig for the above AKS Cluster
		k8sProvider, err := kubernetes.NewProvider(ctx, "aksK8sProvider", &kubernetes.ProviderArgs{
			Kubeconfig: pulumi.String(kubeconfig),
		})
		if err != nil {
			return fmt.Errorf("error whilst getting an existing kubeconfig file: %v", err)
		}
		ctx.Export("aksK8sProvider", k8sProvider.ID())

		return nil
	})
}
