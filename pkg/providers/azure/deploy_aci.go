package azure

import (
	"context"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Uitware/locreg/pkg/parser"
	"log"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerinstance/armcontainerinstance/v2"
)

// DeployACI handles the deployment of an Azure Container Instance
func DeployACI(ctx context.Context, azureConfig *parser.Config, tunnelURL string, envVars map[string]string) {
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
	containerInstance, err := createACI(ctx, azureConfig, tunnelURL, envVars)
	if err != nil {
		cleanupResources(ctx, tracker)
		handleAzureError(err)
	} else {
		tracker.ContainterInstance = azureConfig.Deploy.Provider.Azure.ContainerInstance.Name
		log.Println("✅ Deployment completed successfully.", *containerInstance.Name)
	}

	err = writeProfileContainerInstance(azureConfig.Deploy.Provider.Azure.ResourceGroup, azureConfig.Deploy.Provider.Azure.ContainerInstance.Name)
	if err != nil {
		cleanupResources(ctx, tracker)
		log.Fatal(err)
	}
}

// createACI creates a new Azure Container Instance
func createACI(ctx context.Context, azureConfig *parser.Config, tunnelURL string, envVars map[string]string) (*armcontainerinstance.ContainerGroup, error) {
	containerConfig := azureConfig.Deploy.Provider.Azure.ContainerInstance
	imageConfig := azureConfig.Image

	// Set up environment variables for the container
	envVarsList := []*armcontainerinstance.EnvironmentVariable{}
	for key, value := range envVars {
		envVarsList = append(envVarsList, &armcontainerinstance.EnvironmentVariable{
			Name:  to.Ptr(key),
			Value: to.Ptr(value),
		})
	}

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
								Port: to.Ptr(int32(containerConfig.IPAddress.Ports[0].Port)),
							},
						},
						Resources: &armcontainerinstance.ResourceRequirements{
							Requests: &armcontainerinstance.ResourceRequests{
								CPU:        to.Ptr[float64](containerConfig.Resources.Requests.CPU),
								MemoryInGB: to.Ptr[float64](containerConfig.Resources.Requests.Memory),
							},
						},
						EnvironmentVariables: envVarsList,
					},
				},
			},
			OSType:        to.Ptr(armcontainerinstance.OperatingSystemTypes(containerConfig.OsType)),
			RestartPolicy: to.Ptr(armcontainerinstance.ContainerGroupRestartPolicy(containerConfig.RestartPolicy)),
			IPAddress: &armcontainerinstance.IPAddress{
				Type: to.Ptr(armcontainerinstance.ContainerGroupIPAddressType(containerConfig.IPAddress.Type)),
				Ports: []*armcontainerinstance.Port{
					{
						Port:     to.Ptr(int32(containerConfig.IPAddress.Ports[0].Port)),
						Protocol: to.Ptr(armcontainerinstance.ContainerGroupNetworkProtocol(containerConfig.IPAddress.Ports[0].Protocol)),
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
	profilePath, err := parser.GetProfilePath()
	if err != nil {
		return fmt.Errorf("❌ failed to get profile path: %w", err)
	}

	profile, err := parser.LoadOrCreateProfile(profilePath)
	if err != nil {
		return fmt.Errorf("❌ failed to load or create profile: %w", err)
	}

	if profile.CloudResource == nil {
		profile.CloudResource = &parser.CloudResource{}
	}

	profile.CloudResource.ContainerInstance = &parser.ContainerInstance{
		ResourceGroupName:     resourceGroupName,
		ContainerInstanceName: containerInstanceName,
	}

	if err := parser.SaveProfile(profile, profilePath); err != nil {
		return fmt.Errorf("❌ failed to save profile: %w", err)
	}
	return nil
}
