package aws

import (
	"context"
	"fmt"
	"github.com/Uitware/locreg/pkg/parser"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"log"
	"strconv"
	"strings"
)

type EcsClient struct {
	client       *ecs.Client
	locregConfig *parser.Config
}

// deployECS creates an ECS cluster on Fargate with VPC and public subnet
// that is used to deploy the containers into
func (ecsClient EcsClient) deployECS(ctx context.Context, cfg aws.Config, envVars map[string]string) string {
	profile, _ := parser.LoadProfileData()
	resp, err := ecsClient.client.CreateCluster(ctx, &ecs.CreateClusterInput{
		CapacityProviders: []string{"FARGATE"},
		ClusterName:       aws.String(ecsClient.locregConfig.Deploy.Provider.AWS.ECS.ClusterName),
		Tags:              ecsClient.locregConfig.GenerateECSTags(),
	})
	if err != nil {
		defer Destroy(ecsClient.locregConfig)
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
	IamInstance := IamClient{
		client:       iam.NewFromConfig(cfg),
		locregConfig: ecsClient.locregConfig,
	}
	SecretInstance := SecretsManagerClient{
		client:       secretsmanager.NewFromConfig(cfg),
		locregConfig: ecsClient.locregConfig,
	}
	SecretInstance.createSecret(ctx, profile)
	IamInstance.createRole(ctx, profile)
	subnetId := ec2Instance.createPublicSubnet(ctx, profile)

	// Create task definition
	ecsClient.createTaskDefinition(ctx, profile, envVars)

	log.Println("cluster created ")
	return subnetId
}

func (ecsClient EcsClient) createTaskDefinition(ctx context.Context, profile *parser.Profile, envVars map[string]string) {
	taskRuntimePlatform := types.RuntimePlatform{
		CpuArchitecture:       types.CPUArchitectureX8664,
		OperatingSystemFamily: types.OSFamilyLinux,
	}

	// Prepare environment variables from the env file
	var ECSEnvVars []types.KeyValuePair
	for key, value := range envVars {
		ECSEnvVars = append(ECSEnvVars, types.KeyValuePair{
			Name:  aws.String(key),
			Value: aws.String(value),
		})
	}

	containerDefinition := []types.ContainerDefinition{{
		Name: aws.String(ecsClient.locregConfig.Deploy.Provider.AWS.ECS.TaskDefinition.ContainerDefinition.Name),
		Image: aws.String(fmt.Sprintf("%s/%s",
			strings.TrimPrefix(profile.Tunnel.URL, "https://"),
			ecsClient.locregConfig.GetRegistryImage())),
		RepositoryCredentials: &types.RepositoryCredentials{
			CredentialsParameter: aws.String(profile.AWSCloudResource.ECS.SecretARN),
		},
		PortMappings: ecsClient.locregConfig.GenerateContainerPorts(),
		Environment:  ECSEnvVars,
	}}
	resp, err := ecsClient.client.RegisterTaskDefinition(ctx, &ecs.RegisterTaskDefinitionInput{
		Family:               aws.String(ecsClient.locregConfig.Deploy.Provider.AWS.ECS.TaskDefinition.Family),
		ContainerDefinitions: containerDefinition,
		Cpu:                  aws.String(strconv.Itoa(ecsClient.locregConfig.Deploy.Provider.AWS.ECS.TaskDefinition.CPUAllocation)),
		Memory:               aws.String(strconv.Itoa(ecsClient.locregConfig.Deploy.Provider.AWS.ECS.TaskDefinition.MemoryAllocation)),
		NetworkMode:          types.NetworkModeAwsvpc,
		// Role that allows ECS to pull the image from ECR
		ExecutionRoleArn: aws.String(profile.AWSCloudResource.ECS.RoleARN),
		// For Fargate launch type only
		RuntimePlatform: &taskRuntimePlatform,
		Tags:            ecsClient.locregConfig.GenerateECSTags(),
	})
	if err != nil {
		log.Print("failed to create task definition, " + err.Error())
		defer Destroy(ecsClient.locregConfig)
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
		defer Destroy(ecsClient.locregConfig)
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
		log.Print("failed to destroy task definition, " + err.Error())
	}
	profile.AWSCloudResource.ECS.TaskDefARN = ""
	profile.Save()
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
	profile.AWSCloudResource.ECS.ServiceARN = ""
	profile.Save()
}

func (ecsClient EcsClient) deregisterContainerInstances(ctx context.Context, profile *parser.Profile) {
	// List all container instances in the cluster
	listResp, err := ecsClient.client.ListContainerInstances(ctx, &ecs.ListContainerInstancesInput{
		Cluster: aws.String(profile.AWSCloudResource.ECS.ECSClusterARN),
	})
	if err != nil {
		log.Print("failed to list container instances, " + err.Error())
	}
	if listResp == nil {
		return
	}

	if len(listResp.ContainerInstanceArns) == 0 {
		return
	}
	for _, containerInstance := range listResp.ContainerInstanceArns {
		_, err = ecsClient.client.DeregisterContainerInstance(ctx, &ecs.DeregisterContainerInstanceInput{
			Cluster:           aws.String(profile.AWSCloudResource.ECS.ECSClusterARN),
			ContainerInstance: aws.String(containerInstance),
			Force:             aws.Bool(true),
		})
		if err != nil {
			log.Print("failed to destroy container instance, " + err.Error())
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
	profile.AWSCloudResource.ECS.ECSClusterARN = ""
	profile.Save()
}
