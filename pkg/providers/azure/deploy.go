package azure

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appservice/armappservice/v2"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
	"locreg/pkg/parser"
)

var (
	resourcesClientFactory  *armresources.ClientFactory
	appserviceClientFactory *armappservice.ClientFactory
	resourceGroupClient     *armresources.ResourceGroupsClient
	plansClient             *armappservice.PlansClient
	webAppsClient           *armappservice.WebAppsClient
)

func Deploy(azureConfig *parser.Config) {
	log.Println("Starting deployment...")
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

	resourceGroup, err := createResourceGroup(ctx, azureConfig)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Resource group created:", *resourceGroup.ID)

	appServicePlan, err := createAppServicePlan(ctx, azureConfig)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("App service plan created:", *appServicePlan.ID)

	appService, err := createWebApp(ctx, azureConfig, *appServicePlan.ID)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("App service created:", *appService.ID)

	err = writeProfile(azureConfig.Deploy.Provider.Azure.ResourceGroup, azureConfig.Deploy.Provider.Azure.AppServicePlan.Name, azureConfig.Deploy.Provider.Azure.AppService.Name)
	if err != nil {
		log.Fatalf("Failed to write profile: %v", err)
	}
}

func createResourceGroup(ctx context.Context, azureConfig *parser.Config) (*armresources.ResourceGroup, error) {
	log.Println("Creating Resource Group...")
	resourceGroupResp, err := resourceGroupClient.CreateOrUpdate(
		ctx,
		azureConfig.Deploy.Provider.Azure.ResourceGroup,
		armresources.ResourceGroup{
			Location: to.Ptr(azureConfig.Deploy.Provider.Azure.Location),
		},
		nil)
	if err != nil {
		return nil, err
	}
	return &resourceGroupResp.ResourceGroup, nil
}

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
				Tier:     to.Ptr(sku.Tier),
			},
			Properties: &armappservice.PlanProperties{
				Reserved: to.Ptr(azureConfig.Deploy.Provider.Azure.AppServicePlan.PlanProperties.Reserved),
			},
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

func createWebApp(ctx context.Context, azureConfig *parser.Config, appServicePlanID string) (*armappservice.Site, error) {
	log.Println("Creating Web App...")

	siteConfig := azureConfig.Deploy.Provider.Azure.AppService.SiteConfig

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
					LinuxFxVersion: to.Ptr(fmt.Sprintf("DOCKER|%s:%s", siteConfig.DockerImage, siteConfig.Tag)),
					AppSettings: []*armappservice.NameValuePair{
						{
							Name:  to.Ptr("DOCKER_REGISTRY_SERVER_URL"),
							Value: to.Ptr(siteConfig.DockerRegistryServerUrl),
						},
						{
							Name:  to.Ptr("DOCKER_REGISTRY_SERVER_USERNAME"),
							Value: to.Ptr(azureConfig.Registry.Username),
						},
						{
							Name:  to.Ptr("DOCKER_REGISTRY_SERVER_PASSWORD"),
							Value: to.Ptr(azureConfig.Registry.Password),
						},
					},
				},
				HTTPSOnly: to.Ptr(true),
			},
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

func getSubscriptionID() (string, error) {
	cmd := exec.Command("az", "account", "show", "--query", "id", "--output", "json")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	var result string
	if err := json.Unmarshal(output, &result); err != nil {
		return "", err
	}

	return result, nil
}

func writeProfile(resourceGroupName, appServicePlanName, appServiceName string) error {
	profilePath, err := parser.GetProfilePath()
	if err != nil {
		return fmt.Errorf("failed to get profile path: %w", err)
	}

	profile, err := parser.LoadOrCreateProfile(profilePath)
	if err != nil {
		return fmt.Errorf("failed to load or create profile: %w", err)
	}

	profile.CloudResources.ResourceGroupName = resourceGroupName
	profile.CloudResources.AppServicePlanName = appServicePlanName
	profile.CloudResources.AppServiceName = appServiceName

	err = parser.SaveProfile(profile, profilePath)
	if err != nil {
		return fmt.Errorf("failed to save profile: %w", err)
	}

	return nil
}
