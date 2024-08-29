package aws

import (
	"context"
	"github.com/Uitware/locreg/pkg/parser"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"log"
	"strconv"
)

type EcsClient struct {
	client       *ecs.Client
	locregConfig *parser.Config
}

// deployECS creates an ECS cluster on Fargate with VPC and public subnet
// that is used to deploy the containers into
func (ecsClient EcsClient) deployECS(ctx context.Context, cfg aws.Config) string {
	profile, _ := parser.LoadProfileData()
	resp, err := ecsClient.client.CreateCluster(ctx, &ecs.CreateClusterInput{
		CapacityProviders: []string{"FARGATE"},
		ClusterName:       aws.String(ecsClient.locregConfig.Deploy.Provider.AWS.ECS.ClusterName),
		Tags:              ecsClient.locregConfig.GenerateECSTags(),
	})
	if err != nil {
		defer ecsClient.destroyECS(ctx, profile)
		log.Print("failed to create cluster, " + err.Error())
		return ""
	}

	profile.AWSCloudResource = &parser.AWSCloudResource{
		ECS: &parser.ECS{
			ECSClusterARN: *resp.Cluster.ClusterArn,
		},
	}
	profile.Save()

	// Create VPC with public subnet
	ec2Instance := VpcClient{
		client:       ec2.NewFromConfig(cfg),
		locregConfig: ecsClient.locregConfig,
	}
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
		Name:         aws.String(ecsClient.locregConfig.Deploy.Provider.AWS.ECS.TaskDefinition.ContainerDefinition.Name),
		Image:        aws.String(ecsClient.locregConfig.GetRegistryImage()),
		PortMappings: ecsClient.locregConfig.GenerateContainerPorts(),
	}}
	resp, err := ecsClient.client.RegisterTaskDefinition(ctx, &ecs.RegisterTaskDefinitionInput{
		Family:               aws.String(ecsClient.locregConfig.Deploy.Provider.AWS.ECS.TaskDefinition.Family),
		ContainerDefinitions: containerDefinition,
		Cpu:                  aws.String(strconv.Itoa(ecsClient.locregConfig.Deploy.Provider.AWS.ECS.TaskDefinition.CPUAllocation)),
		Memory:               aws.String(strconv.Itoa(ecsClient.locregConfig.Deploy.Provider.AWS.ECS.TaskDefinition.MemoryAllocation)),
		NetworkMode:          types.NetworkModeAwsvpc,
		// For Fargate launch type only
		RuntimePlatform: &taskRuntimePlatform,
		Tags:            ecsClient.locregConfig.GenerateECSTags(),
	})
	if err != nil {
		defer ecsClient.destroyTaskDefinition(ctx, profile)
		log.Print("failed to create task definition, " + err.Error())
		return
	}
	profile.AWSCloudResource.ECS.TaskDefARN = *resp.TaskDefinition.TaskDefinitionArn
	profile.Save()
}

func (ecsClient EcsClient) runService(ctx context.Context, subnetId string) {
	profile, _ := parser.LoadProfileData()
	resp, err := ecsClient.client.CreateService(ctx, &ecs.CreateServiceInput{
		ServiceName:    aws.String(ecsClient.locregConfig.Deploy.Provider.AWS.ECS.ServiceName),
		TaskDefinition: aws.String(profile.AWSCloudResource.ECS.TaskDefARN),
		Cluster:        aws.String(profile.AWSCloudResource.ECS.ECSClusterARN),
		DesiredCount:   aws.Int32(int32(ecsClient.locregConfig.Deploy.Provider.AWS.ECS.ServiceContainerCount)),
		LaunchType:     types.LaunchTypeFargate,
		NetworkConfiguration: &types.NetworkConfiguration{
			AwsvpcConfiguration: &types.AwsVpcConfiguration{
				AssignPublicIp: types.AssignPublicIpEnabled,
				Subnets:        []string{subnetId},
			},
		},
		Tags: ecsClient.locregConfig.GenerateECSTags(),
	})
	if err != nil {
		defer ecsClient.destroyService(ctx, profile)
		log.Print("failed to run task, " + err.Error())
		return
	}
	profile.AWSCloudResource.ECS.ServiceARN = *resp.Service.ServiceArn
	profile.Save()
}

// destroyTaskDefinition destroys the task definition
func (ecsClient EcsClient) destroyTaskDefinition(ctx context.Context, profile *parser.Profile) {
	_, err := ecsClient.client.DeregisterTaskDefinition(ctx, &ecs.DeregisterTaskDefinitionInput{
		TaskDefinition: aws.String(profile.AWSCloudResource.ECS.TaskDefARN),
	})
	if err != nil {
		log.Fatal("failed to destroy task definition, " + err.Error())
	}
}

// destroyService set service desired count to 0 and delete the service
func (ecsClient EcsClient) destroyService(ctx context.Context, profile *parser.Profile) {
	_, err := ecsClient.client.UpdateService(ctx, &ecs.UpdateServiceInput{
		Cluster:      aws.String(profile.AWSCloudResource.ECS.ECSClusterARN),
		Service:      aws.String(profile.AWSCloudResource.ECS.ServiceARN),
		DesiredCount: aws.Int32(0),
	})
	if err != nil {
		log.Print("failed to stop service, " + err.Error())
	}
	_, err = ecsClient.client.DeleteService(ctx, &ecs.DeleteServiceInput{
		Cluster: aws.String(profile.AWSCloudResource.ECS.ECSClusterARN),
		Service: aws.String(profile.AWSCloudResource.ECS.ServiceARN),
	})
	if err != nil {
		log.Print("failed to destroy service, " + err.Error())
	}
}

func (ecsClient EcsClient) deregisterContainerInstances(ctx context.Context, profile *parser.Profile) {
	// List all container instances in the cluster
	listResp, err := ecsClient.client.ListContainerInstances(ctx, &ecs.ListContainerInstancesInput{
		Cluster: aws.String(profile.AWSCloudResource.ECS.ECSClusterARN),
	})
	if err != nil {
		log.Fatal("failed to list container instances, " + err.Error())
	}

	for _, containerInstance := range listResp.ContainerInstanceArns {
		_, err = ecsClient.client.DeregisterContainerInstance(ctx, &ecs.DeregisterContainerInstanceInput{
			Cluster:           aws.String(profile.AWSCloudResource.ECS.ECSClusterARN),
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
			Cluster: aws.String(profile.AWSCloudResource.ECS.ECSClusterARN),
		})
		return err
	})
}
