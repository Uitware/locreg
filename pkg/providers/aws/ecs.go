package aws

import (
	"context"
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
	resp, err := ecsClient.client.CreateCluster(ctx, &ecs.CreateClusterInput{
		CapacityProviders: []string{"FARGATE"},
		ClusterName:       aws.String("locreg-cluster"),
		Tags:              generateECSTags(),
	})
	if err != nil {
		log.Fatal("failed to create cluster, " + err.Error())
	}

	// Create VPC with public subnet
	ec2Instance := VpcClient{client: ec2.NewFromConfig(cfg)}
	subnetId := ec2Instance.createPublicSubnet(ctx)

	// Create task definition
	ecsClient.createTaskDefinition(ctx)

	log.Println("cluster created, " + *resp.Cluster.ClusterName)
	log.Println("subnet created, " + subnetId)
	return subnetId
}

func (ecsClient EcsClient) createTaskDefinition(ctx context.Context) {
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
		log.Fatal("failed to create task definition, " + err.Error())
	}
	log.Println(resp)
}

func (ecsClient EcsClient) runService(ctx context.Context, subnetId string) {
	_, err := ecsClient.client.CreateService(ctx, &ecs.CreateServiceInput{
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
		log.Fatal("failed to run task, " + err.Error())
	}
}
