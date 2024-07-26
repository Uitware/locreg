package azure

import (
	"context"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"log"

	"locreg/pkg/parser"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appservice/armappservice/v2"
)

// DeployAppService handles the deployment of an Azure App Service
func DeployAppService(ctx context.Context, azureConfig *parser.Config, tunnelURL string, envVars map[string]string) {

	err := checkTunnelURLValidity(tunnelURL)
	if err != nil {
		log.Fatal(err)
	}

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

	// Initialize the App Service client
	appserviceClientFactory, err = armappservice.NewClientFactory(subscriptionID, cred, nil)
	if err != nil {
		log.Fatal(err)
	}
	plansClient = appserviceClientFactory.NewPlansClient()
	webAppsClient = appserviceClientFactory.NewWebAppsClient()

	// Create an App Service plan
	appServicePlan, err := createAppServicePlan(ctx, azureConfig)
	if err != nil {
		cleanupResources(ctx, tracker)
		handleAzureError(err)
	} else {
		tracker.AppServicePlan = azureConfig.Deploy.Provider.Azure.AppServicePlan.Name
		log.Println("✅ App service plan created:", *appServicePlan.ID)
	}

	appService, err := createWebApp(ctx, azureConfig, *appServicePlan.ID, tunnelURL, envVars)
	if err != nil {
		cleanupResources(ctx, tracker)
		handleAzureError(err)
	} else {
		tracker.WebApp = azureConfig.Deploy.Provider.Azure.AppService.Name
		log.Println("✅ App service created:", *appService.ID)
	}
	// Write deployment information to the profile
	err = writeProfileAppService(azureConfig.Deploy.Provider.Azure.ResourceGroup, azureConfig.Deploy.Provider.Azure.AppServicePlan.Name, azureConfig.Deploy.Provider.Azure.AppService.Name)
	if err != nil {
		cleanupResources(ctx, tracker)
		log.Fatal(err)
	}
}

// createAppServicePlan creates a new App Service plan in Azure
func createAppServicePlan(ctx context.Context, azureConfig *parser.Config) (*armappservice.Plan, error) {
	log.Println("Creating App Service Plan...")
	sku := azureConfig.Deploy.Provider.Azure.AppServicePlan.Sku
	pollerResp, err := plansClient.BeginCreateOrUpdate(
		ctx,
		azureConfig.Deploy.Provider.Azure.ResourceGroup,
		azureConfig.Deploy.Provider.Azure.AppServicePlan.Name,
		armappservice.Plan{
			Location: to.Ptr(azureConfig.Deploy.Provider.Azure.Location),
			SKU: &armappservice.SKUDescription{
				Name:     to.Ptr(sku.Name),
				Capacity: to.Ptr[int32](int32(sku.Capacity)),
			},
			Properties: &armappservice.PlanProperties{
				Reserved: to.Ptr(azureConfig.Deploy.Provider.Azure.AppServicePlan.PlanProperties.Reserved),
			},
			Tags: azureConfig.Tags,
		},
		nil,
	)
	if err != nil {
		return nil, err
	}
	resp, err := pollerResp.PollUntilDone(ctx, nil)
	if err != nil {
		return nil, err
	}
	return &resp.Plan, nil
}

// createWebApp creates a new Web App in Azure
func createWebApp(ctx context.Context, azureConfig *parser.Config, appServicePlanID, tunnelURL string, envVars map[string]string) (*armappservice.Site, error) {
	log.Println("Creating Web App...")

	siteConfig := azureConfig.Deploy.Provider.Azure.AppService.SiteConfig
	imageConfig := azureConfig.Image

	// Set up app settings for the Docker container
	appSettings := []*armappservice.NameValuePair{
		{
			Name:  to.Ptr("DOCKER_REGISTRY_SERVER_URL"),
			Value: to.Ptr(fmt.Sprintf("https://%s", tunnelURL)),
		},
		{
			Name:  to.Ptr("DOCKER_REGISTRY_SERVER_USERNAME"),
			Value: to.Ptr(azureConfig.Registry.Username),
		},
		{
			Name:  to.Ptr("DOCKER_REGISTRY_SERVER_PASSWORD"),
			Value: to.Ptr(azureConfig.Registry.Password),
		},
	}

	// Add environment variables from envVars to appSettings
	for key, value := range envVars {
		appSettings = append(appSettings, &armappservice.NameValuePair{
			Name:  to.Ptr(key),
			Value: to.Ptr(value),
		})
	}

	// Create or update the Web App
	pollerResp, err := webAppsClient.BeginCreateOrUpdate(
		ctx,
		azureConfig.Deploy.Provider.Azure.ResourceGroup,
		azureConfig.Deploy.Provider.Azure.AppService.Name,
		armappservice.Site{
			Location: to.Ptr(azureConfig.Deploy.Provider.Azure.Location),
			Properties: &armappservice.SiteProperties{
				ServerFarmID: to.Ptr(appServicePlanID),
				SiteConfig: &armappservice.SiteConfig{
					AlwaysOn:       to.Ptr(siteConfig.AlwaysOn),
					LinuxFxVersion: to.Ptr(fmt.Sprintf("DOCKER|%s/%s:%s", tunnelURL, imageConfig.Name, imageConfig.Tag)),
					AppSettings:    appSettings,
				},
				HTTPSOnly: to.Ptr(true),
			},
			Tags: azureConfig.Tags,
		},
		nil,
	)
	if err != nil {
		return nil, err
	}
	resp, err := pollerResp.PollUntilDone(ctx, nil)
	if err != nil {
		return nil, err
	}
	return &resp.Site, nil
}
func writeProfileAppService(resourceGroupName, appServicePlanName, appServiceName string) error {
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
	profile.CloudResources.AppServicePlanName = appServicePlanName
	profile.CloudResources.AppServiceName = appServiceName

	// Save the updated profile
	err = parser.SaveProfile(profile, profilePath)
	if err != nil {
		return fmt.Errorf("❌ failed to save profile: %w", err)
	}

	return nil
}
