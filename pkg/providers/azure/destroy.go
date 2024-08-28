package azure

import (
	"context"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appservice/armappservice/v2"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerinstance/armcontainerinstance/v2"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
	"github.com/Uitware/locreg/pkg/parser"
	"log"
)

func Destroy() {
	log.Println("Starting destruction...")
	subscriptionID, err := getSubscriptionID()
	if err != nil {
		log.Fatal(err)
	}
	if len(subscriptionID) == 0 {
		log.Fatal("❌ AZURE_SUBSCRIPTION_ID is not set.")
	}

	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()

	resourcesClientFactory, err = armresources.NewClientFactory(subscriptionID, cred, nil)
	if err != nil {
		log.Fatal(err)
	}
	resourceGroupClient = resourcesClientFactory.NewResourceGroupsClient()

	appserviceClientFactory, err = armappservice.NewClientFactory(subscriptionID, cred, nil)
	if err != nil {
		log.Fatal(err)
	}
	plansClient = appserviceClientFactory.NewPlansClient()
	webAppsClient = appserviceClientFactory.NewWebAppsClient()

	aciClientFactory, err = armcontainerinstance.NewClientFactory(subscriptionID, cred, nil)
	if err != nil {
		log.Fatal(err)
	}
	aciClient = aciClientFactory.NewContainerGroupsClient()

	profilePath, err := parser.GetProfilePath()
	if err != nil {
		log.Fatal(err)
	}

	profile, err := parser.LoadOrCreateProfile(profilePath)
	if err != nil {
		log.Fatal(err)
	}

	if profile.AzureCloudResource.AppService != nil {
		if profile.AzureCloudResource.AppService.AppServiceName != "" {
			if err := deleteWebApp(ctx, profile.AzureCloudResource.AppService.AppServiceName, profile.AzureCloudResource.AppService.ResourceGroupName); err != nil {
				handleAzureError(err)
			} else {
				log.Println("✅ App service deleted:", profile.AzureCloudResource.AppService.AppServiceName)
			}
		}

		if profile.AzureCloudResource.AppService.AppServicePlanName != "" {
			if err := deleteAppServicePlan(ctx, profile.AzureCloudResource.AppService.AppServicePlanName, profile.AzureCloudResource.AppService.ResourceGroupName); err != nil {
				handleAzureError(err)
			} else {
				log.Println("✅ App service plan deleted:", profile.AzureCloudResource.AppService.AppServicePlanName)
			}
		}

		if profile.AzureCloudResource.AppService.ResourceGroupName != "" {
			if err := deleteResourceGroup(ctx, profile.AzureCloudResource.AppService.ResourceGroupName); err != nil {
				handleAzureError(err)
			} else {
				log.Println("✅ Resource group deleted:", profile.AzureCloudResource.AppService.ResourceGroupName)
			}
		}
	}
	if profile.AzureCloudResource.ContainerInstance != nil {
		if profile.AzureCloudResource.ContainerInstance.ContainerInstanceName != "" {
			if err := deleteContainerInstance(ctx, profile.AzureCloudResource.ContainerInstance.ContainerInstanceName, profile.AzureCloudResource.ContainerInstance.ResourceGroupName); err != nil {
				handleAzureError(err)
			} else {
				log.Println("✅ Container instance deleted:", profile.AzureCloudResource.ContainerInstance.ContainerInstanceName)
			}
		}

		if profile.AzureCloudResource.ContainerInstance.ResourceGroupName != "" {
			if err := deleteResourceGroup(ctx, profile.AzureCloudResource.ContainerInstance.ResourceGroupName); err != nil {
				handleAzureError(err)
			} else {
				log.Println("✅ Resource group deleted:", profile.AzureCloudResource.ContainerInstance.ResourceGroupName)
			}
		}
	}
}

func deleteResourceGroup(ctx context.Context, resourceGroupName string) error {
	log.Println("Deleting Resource Group...")
	pollerResp, err := resourceGroupClient.BeginDelete(ctx, resourceGroupName, nil)
	if err != nil {
		return err
	}
	_, err = pollerResp.PollUntilDone(ctx, nil)
	if err != nil {
		return err
	}
	return nil
}

func deleteAppServicePlan(ctx context.Context, appServicePlanName, resourceGroupName string) error {
	log.Println("Deleting App Service Plan...")
	_, err := plansClient.Delete(ctx, resourceGroupName, appServicePlanName, nil)
	if err != nil {
		return err
	}
	return nil
}

func deleteWebApp(ctx context.Context, appServiceName, resourceGroupName string) error {
	log.Println("Deleting Web App...")
	_, err := webAppsClient.Delete(ctx, resourceGroupName, appServiceName, nil)
	if err != nil {
		return err
	}
	return nil
}

func deleteContainerInstance(ctx context.Context, containerInstanceName, resourceGroupName string) error {
	log.Println("Deleting Container Instance...")
	pollerResp, err := aciClient.BeginDelete(ctx, resourceGroupName, containerInstanceName, nil)
	if err != nil {
		return err
	}
	_, err = pollerResp.PollUntilDone(ctx, nil)
	if err != nil {
		return err
	}
	return nil
}
