package main

import (
	kubernetesingressnginx "github.com/dirien/pulumi-kubernetes-ingress-nginx/sdk/go/kubernetes-ingress-nginx"
	v1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/apps/v1"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/meta/v1"
	v12 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/networking/v1"

	"github.com/pulumi/pulumi-docker/sdk/v3/go/docker"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {

		ingressController, err := kubernetesingressnginx.NewIngressController(ctx, "nginx-ingress", &kubernetesingressnginx.IngressControllerArgs{
			Controller: kubernetesingressnginx.ControllerArgs{},
		})
		if err != nil {
			return err
		}

		image, err := docker.NewImage(ctx, "my-image", &docker.ImageArgs{
			Build: docker.DockerBuildArgs{
				Context: pulumi.String("./myApp"),
			},
			ImageName: pulumi.String("docker.io/dirien/my-image:latest"),
			Registry:  docker.ImageRegistryArgs{},
		})
		if err != nil {
			return err
		}

		namespace, err := corev1.NewNamespace(ctx, "my-namespace", &corev1.NamespaceArgs{
			Metadata: metav1.ObjectMetaArgs{
				Name: pulumi.String("dummy-ns"),
			},
		})
		if err != nil {
			return err
		}

		v1.NewDeployment(ctx, "dummy-deployment", &v1.DeploymentArgs{
			Metadata: metav1.ObjectMetaArgs{
				Name:      pulumi.String("dummy-deployment"),
				Namespace: namespace.Metadata.Name(),
			},
			Spec: v1.DeploymentSpecArgs{
				Selector: metav1.LabelSelectorArgs{
					MatchLabels: pulumi.StringMap{
						"app": pulumi.String("dummy"),
					},
				},
				Replicas: pulumi.Int(1),
				Template: corev1.PodTemplateSpecArgs{
					Metadata: metav1.ObjectMetaArgs{
						Labels: pulumi.StringMap{
							"app": pulumi.String("dummy"),
						},
					},
					Spec: corev1.PodSpecArgs{
						Containers: corev1.ContainerArray{
							corev1.ContainerArgs{
								Name:  pulumi.String("dummy"),
								Image: image.ImageName,
								Ports: corev1.ContainerPortArray{
									corev1.ContainerPortArgs{
										ContainerPort: pulumi.Int(8080),
									},
								},
								Env: corev1.EnvVarArray{
									corev1.EnvVarArgs{
										Name:  pulumi.String("MESSAGE"),
										Value: pulumi.String("Hello from Pulumi"),
									},
								},
							},
						},
					},
				},
			},
		}, pulumi.DependsOn([]pulumi.Resource{namespace}))

		service, err := corev1.NewService(ctx, "dummy-service", &corev1.ServiceArgs{
			Metadata: metav1.ObjectMetaArgs{
				Name:      pulumi.String("dummy-service"),
				Namespace: namespace.Metadata.Name(),
			},
			Spec: corev1.ServiceSpecArgs{
				Selector: pulumi.StringMap{
					"app": pulumi.String("dummy"),
				},
				Ports: corev1.ServicePortArray{
					corev1.ServicePortArgs{
						Port:       pulumi.Int(80),
						TargetPort: pulumi.Int(8080),
					},
				},
				Type: pulumi.String("ClusterIP"),
			},
		}, pulumi.DependsOn([]pulumi.Resource{namespace}))
		if err != nil {
			return err
		}

		serviceName := service.Metadata.Name().ApplyT(func(name *string) string {
			return *name
		}).(pulumi.StringInput)

		v12.NewIngress(ctx, "dummy-ingress", &v12.IngressArgs{
			Metadata: metav1.ObjectMetaArgs{
				Name: pulumi.String("dummy-ingress"),
			},
			Spec: v12.IngressSpecArgs{
				Rules: v12.IngressRuleArray{
					v12.IngressRuleArgs{
						Host: pulumi.String("dummy"),
						Http: v12.HTTPIngressRuleValueArgs{
							Paths: v12.HTTPIngressPathArray{
								v12.HTTPIngressPathArgs{
									Path:     pulumi.String("/"),
									PathType: pulumi.String("ImplementationSpecific"),
									Backend: v12.IngressBackendArgs{
										Service: v12.IngressServiceBackendArgs{
											Name: serviceName,
											Port: v12.ServiceBackendPortArgs{
												Number: pulumi.Int(80),
											},
										},
									},
								},
							},
						},
					},
				},
			},
		}, pulumi.DependsOn([]pulumi.Resource{namespace, ingressController}))
		return nil
	})
}
