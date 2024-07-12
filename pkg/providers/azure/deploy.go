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

	"locreg/pkg/parser"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appservice/armappservice/v2"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
	"github.com/cenkalti/backoff/v4"
)

var (
	resourcesClientFactory  *armresources.ClientFactory
	appserviceClientFactory *armappservice.ClientFactory
	resourceGroupClient     *armresources.ResourceGroupsClient
	plansClient             *armappservice.PlansClient
	webAppsClient           *armappservice.WebAppsClient
)

type ResourceTracker struct {
	ResourceGroup  string
	AppServicePlan string
	WebApp         string
}

// Deploy initiates the deployment of resources in Azure
func Deploy(azureConfig *parser.Config) {
	log.Println("Starting deployment...")

	// Get the Azure subscription ID
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
	ctx := context.Background()

	// Load the user profile to get the tunnel URL
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

	// Initialize Azure resource clients
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

	// Track created resources
	tracker := &ResourceTracker{}

	// Create a resource group
	resourceGroup, err := createResourceGroup(ctx, azureConfig)
	if err != nil {
		cleanupResources(ctx, tracker)
		log.Fatal(err)
	}
	tracker.ResourceGroup = azureConfig.Deploy.Provider.Azure.ResourceGroup
	log.Println("✅ Resource group created:", *resourceGroup.ID)

	// Create an App Service plan
	appServicePlan, err := createAppServicePlan(ctx, azureConfig)
	if err != nil {
		cleanupResources(ctx, tracker)
		log.Fatal(err)
	}
	tracker.AppServicePlan = azureConfig.Deploy.Provider.Azure.AppServicePlan.Name
	log.Println("✅ App service plan created:", *appServicePlan.ID)

	// Create a Web App
	appService, err := createWebApp(ctx, azureConfig, *appServicePlan.ID, tunnelURL)
	if err != nil {
		cleanupResources(ctx, tracker)
		log.Fatal(err)
	}
	tracker.WebApp = azureConfig.Deploy.Provider.Azure.AppService.Name
	log.Println("✅ App service created:", *appService.ID)

	// Write deployment information to the profile
	err = writeProfile(azureConfig.Deploy.Provider.Azure.ResourceGroup, azureConfig.Deploy.Provider.Azure.AppServicePlan.Name, azureConfig.Deploy.Provider.Azure.AppService.Name)
	if err != nil {
		cleanupResources(ctx, tracker)
		log.Fatalf("❌ Failed to write profile: %v", err)
	}
}

// createResourceGroup creates a new resource group in Azure
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

// createWebApp creates a new Web App in Azure
func createWebApp(ctx context.Context, azureConfig *parser.Config, appServicePlanID, tunnelURL string) (*armappservice.Site, error) {
	log.Println("Creating Web App...")

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

	// Set up app settings for the Docker container
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

// getSubscriptionID retrieves the Azure subscription ID from the Azure CLI
func getSubscriptionID() (string, error) {
	// Execute the Azure CLI command to get the subscription ID
	cmd := exec.Command("az", "account", "show", "--query", "id", "--output", "json")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	// Parse the JSON output to extract the subscription ID
	var result string
	if err := json.Unmarshal(output, &result); err != nil {
		return "", err
	}

	return result, nil
}

// writeProfile updates the profile with the created Azure resources' details
func writeProfile(resourceGroupName, appServicePlanName, appServiceName string) error {
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

// checkTunnelURLValidity checks the validity of the tunnel URL using exponential backoff
func checkTunnelURLValidity(tunnelURL string) error {
	// Define the operation to check the tunnel URL
	operation := func() error {
		// Check if tunnel URL is empty
		if tunnelURL == "" {
			log.Println("❌ Tunnel URL is empty, retrying...")
			return fmt.Errorf("tunnel URL is empty")
		}

		// Make an HTTP GET request to the tunnel URL
		checkURL := fmt.Sprintf("https://%s", tunnelURL)
		resp, err := http.Get(checkURL)
		if err != nil {
			log.Printf("❌ Error checking tunnel URL %s: %v", checkURL, err)
			return err
		}
		defer resp.Body.Close()

		// Check if the response status is OK
		if resp.StatusCode != http.StatusOK {
			log.Printf("❌ Invalid response status for URL %s: %s, retrying...", checkURL, resp.Status)
			return fmt.Errorf("invalid response status: %s", resp.Status)
		}

		log.Printf("✅ Tunnel URL %s is valid", checkURL)
		return nil
	}

	// Generate exponential backoff intervals
	maxRetries := 5
	initialInterval := 1 * time.Second
	intervals := make([]time.Duration, maxRetries)
	for i := 0; i < maxRetries; i++ {
		intervals[i] = initialInterval * (1 << i)
	}

	currentRetry := 0

	// Use backoff.RetryNotify to retry the operation with exponential backoff
	err := backoff.RetryNotify(operation, backoff.WithMaxRetries(backoff.NewConstantBackOff(1*time.Second), uint64(maxRetries)), func(err error, duration time.Duration) {
		if currentRetry < maxRetries {
			duration = intervals[currentRetry]
			currentRetry++
		}
		log.Printf("Retrying in %s due to error: %v", duration, err)
		time.Sleep(duration)
	})
	if err != nil {
		return fmt.Errorf("❌ Tunnel URL check failed after retries: %w", err)
	}

	return nil
}

// cleanupResources deletes all created resources if deployment fails
func cleanupResources(ctx context.Context, tracker *ResourceTracker) {
	log.Println("Cleaning up resources...")
	if tracker.WebApp != "" {
		log.Printf("Deleting Web App: %s...", tracker.WebApp)
		_, err := webAppsClient.Delete(ctx, tracker.ResourceGroup, tracker.WebApp, nil)
		if err != nil {
			log.Printf("❌ Failed to delete Web App: %v", err)
		} else {
			log.Printf("✅ Web App deleted: %s", tracker.WebApp)
		}
	}
	if tracker.AppServicePlan != "" {
		log.Printf("Deleting App Service Plan: %s...", tracker.AppServicePlan)
		_, err := plansClient.Delete(ctx, tracker.ResourceGroup, tracker.AppServicePlan, nil)
		if err != nil {
			log.Printf("❌ Failed to delete App Service Plan: %v", err)
		} else {
			log.Printf("✅ App Service Plan deleted: %s", tracker.AppServicePlan)
		}
	}
	if tracker.ResourceGroup != "" {
		log.Printf("Deleting Resource Group: %s...", tracker.ResourceGroup)
		_, err := resourceGroupClient.BeginDelete(ctx, tracker.ResourceGroup, nil)
		if err != nil {
			log.Printf("❌ Failed to delete Resource Group: %v", err)
		} else {
			log.Printf("✅ Resource Group deletion initiated: %s", tracker.ResourceGroup)
		}
	}
}
