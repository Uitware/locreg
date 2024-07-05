package azure

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appservice/armappservice/v2"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
	"github.com/cenkalti/backoff/v4"
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
	log.Println("☁️ Starting deployment...")
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

	// Load profile to get tunnel URL
	profilePath, err := parser.GetProfilePath()
	if err != nil {
		log.Fatalf("❌ Failed to get profile path: %v", err)
	}

	profile, err := parser.LoadOrCreateProfile(profilePath)
	if err != nil {
		log.Fatalf("❌ Failed to load or create profile: %v", err)
	}

	// Remove 'https://' prefix from the tunnel URL
	tunnelURL := strings.TrimPrefix(profile.Tunnel.URL, "https://")

	// Check the validity of the tunnel URL with exponential backoff
	err = checkTunnelURLValidity(tunnelURL)
	if err != nil {
		log.Fatalf("❌ Failed to validate tunnel URL: %v", err)
	}

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
	log.Println("✅ Resource group created:", *resourceGroup.ID)

	appServicePlan, err := createAppServicePlan(ctx, azureConfig)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("✅ App service plan created:", *appServicePlan.ID)

	appService, err := createWebApp(ctx, azureConfig, *appServicePlan.ID, tunnelURL)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("✅ App service created:", *appService.ID)

	err = writeProfile(azureConfig.Deploy.Provider.Azure.ResourceGroup, azureConfig.Deploy.Provider.Azure.AppServicePlan.Name, azureConfig.Deploy.Provider.Azure.AppService.Name)
	if err != nil {
		log.Fatalf("❌ Failed to write profile: %v", err)
	}
}

func createResourceGroup(ctx context.Context, azureConfig *parser.Config) (*armresources.ResourceGroup, error) {
	log.Println("☁️ Creating Resource Group...")
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
	log.Println("☁️ Creating App Service Plan...")
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

func createWebApp(ctx context.Context, azureConfig *parser.Config, appServicePlanID, tunnelURL string) (*armappservice.Site, error) {
	log.Println("☁️ Creating Web App...")

	siteConfig := azureConfig.Deploy.Provider.Azure.AppService.SiteConfig
	imageConfig := azureConfig.Image

	profilePath, err := parser.GetProfilePath()
	if err != nil {
		return nil, fmt.Errorf("❌ failed to get profile path: %w", err)
	}

	profile, err := parser.LoadOrCreateProfile(profilePath)
	if err != nil {
		return nil, fmt.Errorf("❌ failed to load or create profile: %w", err)
	}

	appSettings := []*armappservice.NameValuePair{
		{
			Name:  to.Ptr("DOCKER_REGISTRY_SERVER_URL"),
			Value: to.Ptr(fmt.Sprintf("https://%s", tunnelURL)),
		},
		{
			Name:  to.Ptr("DOCKER_REGISTRY_SERVER_USERNAME"),
			Value: to.Ptr(profile.LocalRegistry.Username),
		},
		{
			Name:  to.Ptr("DOCKER_REGISTRY_SERVER_PASSWORD"),
			Value: to.Ptr(profile.LocalRegistry.Password),
		},
	}

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
		return fmt.Errorf("❌ failed to get profile path: %w", err)
	}

	profile, err := parser.LoadOrCreateProfile(profilePath)
	if err != nil {
		return fmt.Errorf("❌ failed to load or create profile: %w", err)
	}

	profile.CloudResources.ResourceGroupName = resourceGroupName
	profile.CloudResources.AppServicePlanName = appServicePlanName
	profile.CloudResources.AppServiceName = appServiceName

	err = parser.SaveProfile(profile, profilePath)
	if err != nil {
		return fmt.Errorf("❌ failed to save profile: %w", err)
	}

	return nil
}

func checkTunnelURLValidity(tunnelURL string) error {
	operation := func() error {
		// Check if tunnel URL is empty
		if tunnelURL == "" {
			log.Println("❌ Tunnel URL is empty, retrying...")
			return fmt.Errorf("tunnel URL is empty")
		}

		checkURL := fmt.Sprintf("https://%s", tunnelURL)
		resp, err := http.Get(checkURL)
		if err != nil {
			log.Printf("❌ Error checking tunnel URL %s: %v", checkURL, err)
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			log.Printf("❌ Invalid response status: %s, retrying...", resp.Status)
			return fmt.Errorf("invalid response status: %s", resp.Status)
		}

		log.Printf("✅ Tunnel URL %s is valid", checkURL)
		return nil
	}

	// Create an exponential backoff with custom intervals
	backOff := backoff.NewExponentialBackOff()
	backOff.InitialInterval = 3 * time.Second
	backOff.Multiplier = 2
	backOff.MaxElapsedTime = 2 * time.Minute // Max time to wait

	err := backoff.Retry(operation, backOff)
	if err != nil {
		return fmt.Errorf("❌ Tunnel URL check failed after retries: %w", err)
	}

	return nil
}
