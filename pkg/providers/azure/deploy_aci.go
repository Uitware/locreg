package azure

import (
	"context"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"locreg/pkg/parser"
	"log"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerinstance/armcontainerinstance/v2"
)

// DeployACI handles the deployment of an Azure Container Instance
func DeployACI(ctx context.Context, azureConfig *parser.Config, tunnelURL string) {

	checkTunnelURLValidity(tunnelURL)

	tracker := &ResourceTracker{}

	subscriptionID, err := getSubscriptionID()
	if err != nil {
		log.Fatal(err)
	}
	if len(subscriptionID) == 0 {
		log.Fatal("❌ AZURE_SUBSCRIPTION_ID is not set.")
	}

	// Authenticate using Azure credentials
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		log.Fatal(err)
	}
	// Initialize the Container Groups client
	aciClientFactory, err = armcontainerinstance.NewClientFactory(subscriptionID, cred, nil)
	if err != nil {
		log.Fatal(err)
	}
	aciClient = aciClientFactory.NewContainerGroupsClient()

	// Create a Container Instance
	containerInstance, err := createACI(ctx, azureConfig, tunnelURL)
	if err != nil {
		cleanupResources(ctx, tracker)
		handleAzureError(err)
	}

	log.Println("✅ Deployment completed successfully.", *containerInstance.ID)

	err = writeProfileContainerInstance(azureConfig.Deploy.Provider.Azure.ResourceGroup, azureConfig.Deploy.Provider.Azure.ContainerInstance.Name)
	if err != nil {
		cleanupResources(ctx, tracker)
		log.Fatal(err)
	}
}

// createACI creates a new Azure Container Instance
func createACI(ctx context.Context, azureConfig *parser.Config, tunnelURL string) (*armcontainerinstance.ContainerGroup, error) {
	containerConfig := azureConfig.Deploy.Provider.Azure.ContainerInstance
	imageConfig := azureConfig.Image
	containerGroup := armcontainerinstance.ContainerGroup{
		Location: to.Ptr(azureConfig.Deploy.Provider.Azure.Location),
		Properties: &armcontainerinstance.ContainerGroupPropertiesProperties{
			Containers: []*armcontainerinstance.Container{
				{
					Name: to.Ptr(containerConfig.Name),
					Properties: &armcontainerinstance.ContainerProperties{
						Image: to.Ptr(fmt.Sprintf("%s/%s:%s", tunnelURL, imageConfig.Name, imageConfig.Tag)),

						Ports: []*armcontainerinstance.ContainerPort{
							{
								Port: to.Ptr(int32(containerConfig.IpAddress.Ports[0].Port)),
							},
						},
						Resources: &armcontainerinstance.ResourceRequirements{
							Requests: &armcontainerinstance.ResourceRequests{
								CPU:        to.Ptr[float64](containerConfig.Resources.Requests.Cpu),
								MemoryInGB: to.Ptr[float64](containerConfig.Resources.Requests.Memory),
							},
						},
					},
				},
			},
			OSType:        to.Ptr(armcontainerinstance.OperatingSystemTypes(containerConfig.OsType)),
			RestartPolicy: to.Ptr(armcontainerinstance.ContainerGroupRestartPolicy(containerConfig.RestartPolicy)),
			IPAddress: &armcontainerinstance.IPAddress{
				Type: to.Ptr(armcontainerinstance.ContainerGroupIPAddressType(containerConfig.IpAddress.Type)),
				Ports: []*armcontainerinstance.Port{
					{
						Port:     to.Ptr(int32(containerConfig.IpAddress.Ports[0].Port)),
						Protocol: to.Ptr(armcontainerinstance.ContainerGroupNetworkProtocol(containerConfig.IpAddress.Ports[0].Protocol)),
					},
				},
			},
			ImageRegistryCredentials: []*armcontainerinstance.ImageRegistryCredential{
				{
					Server:   to.Ptr(tunnelURL),
					Username: to.Ptr(azureConfig.Registry.Username),
					Password: to.Ptr(azureConfig.Registry.Password),
				},
			},
		},
		Tags: azureConfig.Tags,
	}

	pollerResp, err := aciClient.BeginCreateOrUpdate(ctx, azureConfig.Deploy.Provider.Azure.ResourceGroup, containerConfig.Name, containerGroup, nil)
	if err != nil {
		return nil, err
	}

	resp, err := pollerResp.PollUntilDone(ctx, nil)
	if err != nil {
		return nil, err
	}

	log.Println("✅ Azure Container Instance created:", *resp.ID)
	return &resp.ContainerGroup, nil
}
func writeProfileContainerInstance(resourceGroupName, containerInstanceName string) error {
	// Get the profile path
	profilePath, err := parser.GetProfilePath()
	if err != nil {
		return fmt.Errorf("❌ failed to get profile path: %w", err)
	}

	// Load or create a new profile
	profile, err := parser.LoadOrCreateProfile(profilePath)
	if err != nil {
		return fmt.Errorf("❌ failed to load or create profile: %w", err)
	}

	// Update the profile with the new resource details
	profile.CloudResources.ResourceGroupName = resourceGroupName
	profile.CloudResources.ContainerInstanceName = containerInstanceName
	// Save the updated profile
	err = parser.SaveProfile(profile, profilePath)
	if err != nil {
		return fmt.Errorf("❌ failed to save profile: %w", err)
	}

	return nil
}
