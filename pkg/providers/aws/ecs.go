package aws

import (
	"context"
	"github.com/Uitware/locreg/pkg/parser"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"log"
)

type EcsClient struct {
	client *ecs.Client
}

func generateECSTags() []types.Tag {
	return []types.Tag{{
		Key:   aws.String("managed-by"),
		Value: aws.String("locreg"),
	}}
}

// deployECS creates an ECS cluster on Fargate with VPC and public subnet
// that is used to deploy the containers into
func (ecsClient EcsClient) deployECS(ctx context.Context, cfg aws.Config) string {
	profile, _ := parser.LoadProfileData()
	resp, err := ecsClient.client.CreateCluster(ctx, &ecs.CreateClusterInput{
		CapacityProviders: []string{"FARGATE"},
		ClusterName:       aws.String("locreg-cluster"),
		Tags:              generateECSTags(),
	})
	if err != nil {
		defer ecsClient.destroyECS(ctx, profile)
		log.Print("failed to create cluster, " + err.Error())
		return ""
	}

	profile.AWSCloudResource = &parser.AWSCloudResource{
		ECSClusterARN: *resp.Cluster.ClusterArn,
	}
	profile.Save()

	// Create VPC with public subnet
	ec2Instance := VpcClient{client: ec2.NewFromConfig(cfg)}
	subnetId := ec2Instance.createPublicSubnet(ctx, profile)

	// Create task definition
	ecsClient.createTaskDefinition(ctx, profile)

	log.Println("cluster created ")
	return subnetId
}

func (ecsClient EcsClient) createTaskDefinition(ctx context.Context, profile *parser.Profile) {
	taskRuntimePlatform := types.RuntimePlatform{
		CpuArchitecture:       types.CPUArchitectureX8664,
		OperatingSystemFamily: types.OSFamilyLinux,
	}
	containerDefinition := []types.ContainerDefinition{{
		Name:  aws.String("locreg-container"),
		Image: aws.String("nginx"),
		PortMappings: []types.PortMapping{
			{
				ContainerPort: aws.Int32(80),
				HostPort:      aws.Int32(80),
				Protocol:      types.TransportProtocolTcp,
			}},
	}}
	resp, err := ecsClient.client.RegisterTaskDefinition(ctx, &ecs.RegisterTaskDefinitionInput{
		Family:               aws.String("locreg-test-task"),
		ContainerDefinitions: containerDefinition,
		Cpu:                  aws.String("1024"),
		Memory:               aws.String("2048"),
		NetworkMode:          types.NetworkModeAwsvpc,
		// For Fargate launch type only
		RuntimePlatform: &taskRuntimePlatform,
		Tags:            generateECSTags(),
	})
	if err != nil {
		defer ecsClient.destroyTaskDefinition(ctx, profile)
		log.Print("failed to create task definition, " + err.Error())
		return
	}
	profile.AWSCloudResource.TaskDefARN = *resp.TaskDefinition.TaskDefinitionArn
	profile.Save()
}

func (ecsClient EcsClient) runService(ctx context.Context, subnetId string) {
	profile, _ := parser.LoadProfileData()
	resp, err := ecsClient.client.CreateService(ctx, &ecs.CreateServiceInput{
		ServiceName:    aws.String("locreg-service"),
		TaskDefinition: aws.String("locreg-test-task"),
		Cluster:        aws.String("locreg-cluster"),
		DesiredCount:   aws.Int32(1),
		LaunchType:     types.LaunchTypeFargate,
		NetworkConfiguration: &types.NetworkConfiguration{
			AwsvpcConfiguration: &types.AwsVpcConfiguration{
				AssignPublicIp: types.AssignPublicIpEnabled,
				Subnets:        []string{subnetId},
			},
		},
	})
	if err != nil {
		defer ecsClient.destroyService(ctx, profile)
		log.Print("failed to run task, " + err.Error())
		return
	}
	profile.AWSCloudResource.ServiceARN = *resp.Service.ServiceArn
	profile.Save()
}

// destroyTaskDefinition destroys the task definition
func (ecsClient EcsClient) destroyTaskDefinition(ctx context.Context, profile *parser.Profile) {
	_, err := ecsClient.client.DeregisterTaskDefinition(ctx, &ecs.DeregisterTaskDefinitionInput{
		TaskDefinition: aws.String(profile.AWSCloudResource.TaskDefARN),
	})
	if err != nil {
		log.Fatal("failed to destroy task definition, " + err.Error())
	}
}

// destroyService set service desired count to 0 and delete the service
func (ecsClient EcsClient) destroyService(ctx context.Context, profile *parser.Profile) {
	_, err := ecsClient.client.UpdateService(ctx, &ecs.UpdateServiceInput{
		Cluster:      aws.String(profile.AWSCloudResource.ECSClusterARN),
		Service:      aws.String(profile.AWSCloudResource.ServiceARN),
		DesiredCount: aws.Int32(0),
	})
	if err != nil {
		log.Print("failed to stop service, " + err.Error())
	}
	_, err = ecsClient.client.DeleteService(ctx, &ecs.DeleteServiceInput{
		Cluster: aws.String("locreg-cluster"),
		Service: aws.String("locreg-service"),
	})
	if err != nil {
		log.Print("failed to destroy service, " + err.Error())
	}
}

func (ecsClient EcsClient) deregisterContainerInstances(ctx context.Context, profile *parser.Profile) {
	// List all container instances in the cluster
	listResp, err := ecsClient.client.ListContainerInstances(ctx, &ecs.ListContainerInstancesInput{
		Cluster: aws.String(profile.AWSCloudResource.ECSClusterARN),
	})
	if err != nil {
		log.Fatal("failed to list container instances, " + err.Error())
	}

	for _, containerInstance := range listResp.ContainerInstanceArns {
		_, err = ecsClient.client.DeregisterContainerInstance(ctx, &ecs.DeregisterContainerInstanceInput{
			Cluster:           aws.String(profile.AWSCloudResource.ECSClusterARN),
			ContainerInstance: aws.String(containerInstance),
			Force:             aws.Bool(true),
		})
		if err != nil {
			log.Fatal("failed to destroy container instance, " + err.Error())
		}
	}
}

// destroyECS deregister container instances and destroys the ECS cluster
func (ecsClient EcsClient) destroyECS(ctx context.Context, profile *parser.Profile) {
	ecsClient.deregisterContainerInstances(ctx, profile)

	retryOnError(5, 5, func() error {
		_, err := ecsClient.client.DeleteCluster(ctx, &ecs.DeleteClusterInput{
			Cluster: aws.String(profile.AWSCloudResource.ECSClusterARN),
		})
		return err
	})
}
