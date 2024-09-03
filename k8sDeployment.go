package main

import (
	"fmt"
	"os"

	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	appsV1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/apps/v1"
	v1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	metaV1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type Provider struct {
	ctx         *pulumi.Context
	k8sProvider *kubernetes.Provider
	namespace   string
}

func NewProvider(ctx *pulumi.Context, k8sProvider *kubernetes.Provider, namespace string) *Provider {
	return &Provider{
		ctx:         ctx,
		k8sProvider: k8sProvider,
		namespace:   namespace,
	}
}

func (p *Provider) CreatePVC(pvcName string) (*v1.PersistentVolumeClaim, error) {
	pvc, err := v1.NewPersistentVolumeClaim(p.ctx, pvcName, &v1.PersistentVolumeClaimArgs{
		Metadata: &metaV1.ObjectMetaArgs{
			Name:      pulumi.String(pvcName),
			Namespace: pulumi.String(p.namespace),
		},
		Spec: &v1.PersistentVolumeClaimSpecArgs{
			AccessModes: pulumi.StringArray{
				pulumi.String("ReadWriteOnce"),
			},
			Resources: &v1.VolumeResourceRequirementsArgs{
				Requests: pulumi.StringMap{
					"storage": pulumi.String("1Gi"),
				},
			},
		},
	}, pulumi.Provider(p.k8sProvider))
	if err != nil {
		return nil, fmt.Errorf("error with the creation of PVC %s: %v", pvcName, err)
	}
	return pvc, nil
}

func (p *Provider) CreateConfigMap(configName, filePath string) (*v1.ConfigMap, error) {
	configData, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("error reading config file %s: %v", filePath, err)
	}

	configMap, err := v1.NewConfigMap(p.ctx, configName, &v1.ConfigMapArgs{
		Metadata: &metaV1.ObjectMetaArgs{
			Name:      pulumi.String(configName),
			Namespace: pulumi.String(p.namespace),
		},
		Data: pulumi.StringMap{
			"node.conf": pulumi.String(string(configData)),
		},
	}, pulumi.Provider(p.k8sProvider))
	if err != nil {
		return nil, fmt.Errorf("error creating config map %s: %v", configName, err)
	}
	return configMap, nil
}

func (p *Provider) CreateDeployment(deploymentName string, configMap *v1.ConfigMap, pvcNames []string, initializeCommand string) (*appsV1.Deployment, error) {
	volumes := v1.VolumeArray{
		&v1.VolumeArgs{
			Name: pulumi.String(pvcNames[0]),
			PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSourceArgs{
				ClaimName: pulumi.String(pvcNames[0]),
			},
		},
		&v1.VolumeArgs{
			Name: pulumi.String(pvcNames[1]),
			PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSourceArgs{
				ClaimName: pulumi.String(pvcNames[1]),
			},
		},
		&v1.VolumeArgs{
			Name: pulumi.String(pvcNames[2]),
			PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSourceArgs{
				ClaimName: pulumi.String(pvcNames[2]),
			},
		},
		&v1.VolumeArgs{
			Name: pulumi.String(pvcNames[3]),
			PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSourceArgs{
				ClaimName: pulumi.String(pvcNames[3]),
			},
		},
		&v1.VolumeArgs{
			Name: pulumi.String(pvcNames[4]),
			ConfigMap: &v1.ConfigMapVolumeSourceArgs{
				Name: configMap.Metadata.Name(),
			},
		},
	}

	deployment, err := appsV1.NewDeployment(p.ctx, deploymentName, &appsV1.DeploymentArgs{
		Metadata: &metaV1.ObjectMetaArgs{
			Name:      pulumi.String(deploymentName),
			Namespace: pulumi.String(p.namespace),
		},
		Spec: &appsV1.DeploymentSpecArgs{
			Strategy: &appsV1.DeploymentStrategyArgs{
				Type: pulumi.String("Recreate"),
			},
			Selector: &metaV1.LabelSelectorArgs{
				MatchLabels: pulumi.StringMap{
					"run": pulumi.String(deploymentName),
				},
			},
			Template: &v1.PodTemplateSpecArgs{
				Metadata: &metaV1.ObjectMetaArgs{
					Labels: pulumi.StringMap{
						"run": pulumi.String(deploymentName),
					},
				},
				Spec: &v1.PodSpecArgs{
					Containers: v1.ContainerArray{
						&v1.ContainerArgs{
							Name:            pulumi.String(deploymentName),
							Image:           pulumi.String("corda/corda-enterprise:4.12-zulu-openjdk-alpine"),
							ImagePullPolicy: pulumi.String("IfNotPresent"),
							Ports: v1.ContainerPortArray{
								&v1.ContainerPortArgs{
									ContainerPort: pulumi.Int(10005),
									Name:          pulumi.String("p2pport"),
								},
								&v1.ContainerPortArgs{
									ContainerPort: pulumi.Int(10006),
									Name:          pulumi.String("rpcport"),
								},
								&v1.ContainerPortArgs{
									ContainerPort: pulumi.Int(10046),
									Name:          pulumi.String("adminrpcport"),
								},
							},
							Resources: &v1.ResourceRequirementsArgs{
								Limits: pulumi.StringMap{
									"memory": pulumi.String("8Gi"),
									"cpu":    pulumi.String("1"),
								},
								Requests: pulumi.StringMap{
									"memory": pulumi.String("4Gi"),
									"cpu":    pulumi.String("100m"),
								},
							},
							VolumeMounts: v1.VolumeMountArray{
								&v1.VolumeMountArgs{
									Name:      pulumi.String(deploymentName + "-config-pvc"),
									MountPath: pulumi.String("/etc/corda"),
								},
								&v1.VolumeMountArgs{
									Name:      pulumi.String(deploymentName + "-certificates-pvc"),
									MountPath: pulumi.String("/opt/corda/certificates"),
								},
								&v1.VolumeMountArgs{
									Name:      pulumi.String(deploymentName + "-persistence-pvc"),
									MountPath: pulumi.String("/opt/corda/persistence"),
									SubPath:   pulumi.String("persistence"),
								},
								&v1.VolumeMountArgs{
									Name:      pulumi.String(deploymentName + "-persistence-pvc"),
									MountPath: pulumi.String("/opt/corda/artemis"),
									SubPath:   pulumi.String("artemis"),
								},
								&v1.VolumeMountArgs{
									Name:      pulumi.String(deploymentName + "-logs-pvc"),
									MountPath: pulumi.String("/opt/corda/logs"),
								},
								&v1.VolumeMountArgs{
									Name:      pulumi.String(deploymentName + "-configmap-pvc"),
									MountPath: pulumi.String("/opt/corda/config"),
								},
							},

							Env: v1.EnvVarArray{
								&v1.EnvVarArgs{
									Name:  pulumi.String("CORDA_ARGS"),
									Value: pulumi.String("--log-to-console" + initializeCommand),
								},
								&v1.EnvVarArgs{
									Name:  pulumi.String("ACCEPT_LICENSE"),
									Value: pulumi.String("Y"),
								},
								&v1.EnvVarArgs{
									Name:  pulumi.String("CONFIG_FOLDER"),
									Value: pulumi.String("/opt/corda/config"),
								},
							},
						},
					},
					SecurityContext: &v1.PodSecurityContextArgs{
						FsGroup: pulumi.Int(1000),
					},
					Volumes: volumes,
				},
			},
		},
	}, pulumi.Provider(p.k8sProvider))
	if err != nil {
		return nil, fmt.Errorf("error creating deployment %s: %v", deploymentName, err)
	}
	return deployment, nil
}
