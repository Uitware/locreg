package azure

import (
	"context"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appservice/armappservice/v2"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
	"locreg/pkg/parser"
	"log"
)

func Destroy() {
	log.Println("Starting destruction...")
	subscriptionID, err := getSubscriptionID()
	if err != nil {
		log.Fatal(err)
	}
	if len(subscriptionID) == 0 {
		log.Fatal("AZURE_SUBSCRIPTION_ID is not set.")
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

	profilePath, err := parser.GetProfilePath()
	if err != nil {
		log.Fatal(err)
	}

	profile, err := parser.LoadOrCreateProfile(profilePath)
	if err != nil {
		log.Fatal(err)
	}

	if profile.CloudResources.AppServiceName != "" {
		if err := deleteWebApp(ctx, profile.CloudResources.AppServiceName, profile.CloudResources.ResourceGroupName); err != nil {
			log.Printf("Error deleting app service: %v", err)
		} else {
			log.Println("App service deleted:", profile.CloudResources.AppServiceName)
		}
	}

	if profile.CloudResources.AppServicePlanName != "" {
		if err := deleteAppServicePlan(ctx, profile.CloudResources.AppServicePlanName, profile.CloudResources.ResourceGroupName); err != nil {
			log.Printf("Error deleting app service plan: %v", err)
		} else {
			log.Println("App service plan deleted:", profile.CloudResources.AppServicePlanName)
		}
	}

	if profile.CloudResources.ResourceGroupName != "" {
		if err := deleteResourceGroup(ctx, profile.CloudResources.ResourceGroupName); err != nil {
			log.Printf("Error deleting resource group: %v", err)
		} else {
			log.Println("Resource group deleted:", profile.CloudResources.ResourceGroupName)
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
