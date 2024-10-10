package aws

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/service/ecs"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"os"
	"path/filepath"
	"testing"

	"github.com/Uitware/locreg/pkg/parser"
	"github.com/aws/aws-sdk-go-v2/config"
)

// getProjectRoot returns the root directory of the project.
// Adjust the path as necessary based on your project structure.
func getProjectRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		panic("Can't get current directory")
	}
	dir = filepath.Join(dir, "..", "..", "..")
	return dir
}

func TestVPC(t *testing.T) {
	// Load test configuration
	configFilePath := filepath.Join(getProjectRoot(), "test", "test_configs", "aws", "locreg.yaml")
	configFile, err := parser.LoadConfig(configFilePath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	ctx := context.Background()

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		t.Fatalf("Failed to load AWS config: %v", err)
	}

	// Create an EC2 client
	ec2Client := ec2.NewFromConfig(cfg)
	vpcInstance := VpcClient{
		client:       ec2Client,
		locregConfig: configFile,
	}

	// Load profile data
	profile, _ := parser.LoadProfileData()
	profile.AWSCloudResource = &parser.AWSCloudResource{}
	profile.Save()

	vpcInstance.createPublicSubnet(ctx, profile)

	profile, _ = parser.LoadProfileData()
	expectedVPCID, err := getExpectedVPCID(profile)
	if err != nil {
		t.Fatalf("Failed to get expected VPC ID from profile: %v", err)
	}

	actualVPCID, err := getDeployedVPCID(ctx, ec2Client)
	if err != nil {
		t.Fatalf("Failed to retrieve deployed VPC ID: %v", err)
	}

	if actualVPCID != expectedVPCID {
		t.Errorf("VPC ID mismatch: expected %s, got %s", expectedVPCID, actualVPCID)
	} else {
		t.Logf("VPC ID matches expected ID: %s", actualVPCID)
	}

	// Ensure cleanup happens regardless of test outcome
	t.Cleanup(func() {
		vpcInstance.destroyVpc(ctx, profile)
	})
}

// // TestDeployECS tests the deployment of an ECS service and verifies its ARN.
func TestDeployECS(t *testing.T) {
	// Load test configuration
	configFilePath := filepath.Join(getProjectRoot(), "test", "test_configs", "aws", "locreg.yaml")
	configFile, err := parser.LoadConfig(configFilePath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Setup context
	ctx := context.Background()

	// Authenticate using AWS credentials
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		t.Fatalf("Failed to load AWS config: %v", err)
	}

	// Populate profile with mok data
	profile, _ := parser.LoadProfileData()
	profile.LocalRegistry = &parser.LocalRegistry{
		RegistryID: "some-random-id",
		Password:   "123213123",
		Username:   "asdasdasd",
	}
	profile.Tunnel = &parser.Tunnel{
		URL:         "https://docker.io",
		ContainerID: "container-id",
	}
	profile.Save()

	// Create an ECS client
	ecsClient := ecs.NewFromConfig(cfg)
	ecsInstance := EcsClient{client: ecsClient, locregConfig: configFile}

	// Deploy ECS
	ecsInstance.deployECS(ctx, cfg, map[string]string{})

	// Load profile data
	profile, _ = parser.LoadProfileData()
	if profile == nil {
		t.Fatalf("Failed to load profile data")
	}

	t.Logf("%+v", profile.AWSCloudResource.ECS)

	// Retrieve expected ARN from profile
	expectedARN, err := getExpectedECSArn(profile)
	if err != nil {
		t.Fatalf("Failed to get expected ECS ARN from profile: %v", err)
	}

	// Describe the deployed ECS service to get its ARN
	actualARN, err := getDeployedECSClusterArn(ctx, ecsClient, configFile)
	if err != nil {
		t.Fatalf("Failed to retrieve deployed ECS ARN: %v", err)
	}

	// Compare the expected ARN with the actual ARN
	if actualARN != expectedARN {
		t.Errorf("ECS ARN mismatch: expected %s, got %s", expectedARN, actualARN)
	} else {
		t.Logf("ECS ARN matches expected ARN: %s", actualARN)
	}

	// Ensure cleanup happens regardless of test outcome
	t.Cleanup(func() {
		Destroy(configFile)
	})
}

// getExpectedECSArn retrieves the expected ECS ARN from the profile.
// It checks the AWSCloudResource.ECS.ServiceARN field.
// If the ServiceARN is not set, it returns an error.
func getExpectedECSArn(profile *parser.Profile) (string, error) {
	if profile.AWSCloudResource == nil || profile.AWSCloudResource.ECS == nil {
		return "", fmt.Errorf("AWSCloudResource or ECS configuration not found in profile")
	}
	if profile.AWSCloudResource.ECS.ECSClusterARN == "" {
		return "", fmt.Errorf("ECS ServiceARN not found in profile")
	}
	return profile.AWSCloudResource.ECS.ECSClusterARN, nil
}

// getDeployedECSClusterArn describes the ECS cluster and retrieves its ARN.
// It uses the ClusterName from the configFile to identify the cluster.
func getDeployedECSClusterArn(ctx context.Context, client *ecs.Client, configFile *parser.Config) (string, error) {
	// Extract ClusterName from the configFile
	clusterName := configFile.Deploy.Provider.AWS.ECS.ClusterName

	if clusterName == "" {
		return "", fmt.Errorf("ClusterName not specified in configFile")
	}

	describeInput := &ecs.DescribeClustersInput{
		Clusters: []string{clusterName},
	}

	describeOutput, err := client.DescribeClusters(ctx, describeInput)
	if err != nil {
		return "", fmt.Errorf("DescribeClusters API call failed: %w", err)
	}

	if len(describeOutput.Clusters) == 0 {
		return "", fmt.Errorf("no clusters found with name %s", clusterName)
	}

	// Assuming the first cluster is the one deployed
	cluster := describeOutput.Clusters[0]
	if cluster.ClusterArn == nil || *cluster.ClusterArn == "" {
		return "", fmt.Errorf("cluster ARN is empty")
	}

	return *cluster.ClusterArn, nil
}

// getExpectedVPCID retrieves the expected VPC ID from the profile.
// It checks the AWSCloudResource.VPC.VpcID field.
func getExpectedVPCID(profile *parser.Profile) (string, error) {
	if profile.AWSCloudResource == nil || profile.AWSCloudResource.VPC == nil {
		return "", fmt.Errorf("AWSCloudResource or VPC configuration not found in profile")
	}
	if profile.AWSCloudResource.VPC.VPCId == "" {
		return "", fmt.Errorf("VPC ID not found in profile")
	}
	return profile.AWSCloudResource.VPC.VPCId, nil
}

// getDeployedVPCID describes the VPC and retrieves its ID.
// It identifies the VPC based on the "managed-by" tag set to "locreg".
func getDeployedVPCID(ctx context.Context, client *ec2.Client) (string, error) {
	// Describe VPCs
	describeInput := &ec2.DescribeVpcsInput{}
	describeOutput, err := client.DescribeVpcs(ctx, describeInput)
	if err != nil {
		return "", fmt.Errorf("DescribeVpcs API call failed: %w", err)
	}

	// Iterate through VPCs to find one with the "managed-by" tag set to "locreg"
	for _, vpc := range describeOutput.Vpcs {
		if vpc.Tags != nil {
			for _, tag := range vpc.Tags {
				if *tag.Key == "managed-by" && *tag.Value == "locreg" {
					if vpc.VpcId != nil && *vpc.VpcId != "" {
						return *vpc.VpcId, nil
					}
					return "", fmt.Errorf("VPC ID is empty")
				}
			}
		}
	}

	return "", fmt.Errorf("no VPCs found with the 'managed-by' tag set to 'locreg'")
}
