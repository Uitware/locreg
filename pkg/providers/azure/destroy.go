package azure

import (
	"context"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appservice/armappservice/v2"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
	"locreg/pkg/parser"
	"log"
)

func Destroy(azureConfig *parser.Config) {
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

	if err := deleteWebApp(ctx, azureConfig); err != nil {
		log.Printf("Error deleting app service: %v", err)
	} else {
		log.Println("App service deleted:", azureConfig.Deploy.Provider.Azure.AppService.Name)
	}

	if err := deleteAppServicePlan(ctx, azureConfig); err != nil {
		log.Printf("Error deleting app service plan: %v", err)
	} else {
		log.Println("App service plan deleted:", azureConfig.Deploy.Provider.Azure.AppServicePlan.Name)
	}

	if err := deleteResourceGroup(ctx, azureConfig); err != nil {
		log.Printf("Error deleting resource group: %v", err)
	} else {
		log.Println("Resource group deleted:", azureConfig.Deploy.Provider.Azure.ResourceGroup)
	}
}

func deleteResourceGroup(ctx context.Context, azureConfig *parser.Config) error {
	log.Println("Deleting Resource Group...")
	pollerResp, err := resourceGroupClient.BeginDelete(ctx, azureConfig.Deploy.Provider.Azure.ResourceGroup, nil)
	if err != nil {
		return err
	}
	_, err = pollerResp.PollUntilDone(ctx, nil)
	if err != nil {
		return err
	}
	return nil
}

func deleteAppServicePlan(ctx context.Context, azureConfig *parser.Config) error {
	log.Println("Deleting App Service Plan...")
	_, err := plansClient.Delete(ctx, azureConfig.Deploy.Provider.Azure.ResourceGroup, azureConfig.Deploy.Provider.Azure.AppServicePlan.Name, nil)
	if err != nil {
		return err
	}
	return nil
}

func deleteWebApp(ctx context.Context, azureConfig *parser.Config) error {
	log.Println("Deleting Web App...")
	_, err := webAppsClient.Delete(ctx, azureConfig.Deploy.Provider.Azure.ResourceGroup, azureConfig.Deploy.Provider.Azure.AppService.Name, nil)
	if err != nil {
		return err
	}
	return nil
}
