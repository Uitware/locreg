package azure

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appservice/armappservice/v2"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerinstance/armcontainerinstance/v2"
	"github.com/cenkalti/backoff/v4"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"locreg/pkg/parser"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
)

var (
	resourcesClientFactory  *armresources.ClientFactory
	appserviceClientFactory *armappservice.ClientFactory
	aciClientFactory        *armcontainerinstance.ClientFactory

	resourceGroupClient *armresources.ResourceGroupsClient
	plansClient         *armappservice.PlansClient
	webAppsClient       *armappservice.WebAppsClient
	aciClient           *armcontainerinstance.ContainerGroupsClient
)

// ResourceTracker tracks the created resources for cleanup
type ResourceTracker struct {
	ResourceGroup      string
	AppServicePlan     string
	WebApp             string
	ContainterInstance string
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

	// Initialize Azure resource clients
	resourcesClientFactory, err = armresources.NewClientFactory(subscriptionID, cred, nil)
	if err != nil {
		log.Fatal(err)
	}
	resourceGroupClient = resourcesClientFactory.NewResourceGroupsClient()

	// Create a resource group
	resourceGroup, err := createResourceGroup(ctx, azureConfig)
	if err != nil {
		handleAzureError(err)
	}
	log.Println("✅ Resource group created:", *resourceGroup.ID)
	// Fetch the tunnel URL from the profile
	profilePath, err := parser.GetProfilePath()
	if err != nil {
		log.Fatalf("❌ Failed to get profile path: %v", err)
	}

	profile, err := parser.LoadOrCreateProfile(profilePath)
	if err != nil {
		log.Fatalf("❌ Failed to load or create profile: %v", err)
	}
	tunnelURL := strings.TrimPrefix(profile.Tunnel.URL, "https://")
	// Determine the deployment type and call the appropriate deployment function
	if azureConfig.Deploy.Provider.Azure.AppServicePlan.Name != "" {
		DeployAppService(ctx, azureConfig, tunnelURL)
	} else if "azureConfig.Deploy.Provider.Azure.ContainerInstance.Name" != "" {
		DeployACI(ctx, azureConfig, tunnelURL)
	} else {
		log.Fatal("❌ No valid deployment configuration found.")
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
			Tags:     azureConfig.Tags,
		},
		nil)
	if err != nil {
		return nil, err
	}
	return &resourceGroupResp.ResourceGroup, nil
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
		err := deleteWebApp(ctx, tracker.WebApp, tracker.ResourceGroup)
		if err != nil {
			log.Fatalf("❌ Failed to delete Web App: %v", err)
		} else {
			log.Printf("✅ Web App deleted: %s", tracker.WebApp)
		}
	}

	if tracker.AppServicePlan != "" {
		log.Printf("Deleting App Service Plan: %s...", tracker.AppServicePlan)
		err := deleteAppServicePlan(ctx, tracker.AppServicePlan, tracker.ResourceGroup)
		if err != nil {
			log.Fatalf("❌ Failed to delete App Service Plan: %v", err)
		} else {
			log.Printf("✅ App Service Plan deleted: %s", tracker.AppServicePlan)
		}
	}
	if tracker.ContainterInstance != "" {
		log.Printf("Deleting Container Instance: %s...", tracker.ContainterInstance)
		err := deleteContainerInstance(ctx, tracker.ContainterInstance, tracker.ResourceGroup)
		if err != nil {
			log.Fatalf("❌ Failed to delete Container Instance: %v", err)
		} else {
			log.Printf("✅ Container Instance deleted: %s", tracker.ContainterInstance)
		}
	}
	if tracker.ResourceGroup != "" {
		log.Printf("Deleting Resource Group: %s...", tracker.ResourceGroup)
		err := deleteResourceGroup(ctx, tracker.ResourceGroup)
		if err != nil {
			log.Fatalf("❌ Failed to delete Resource Group: %v", err)
		} else {
			log.Printf("✅ Resource Group deletion initiated: %s", tracker.ResourceGroup)
		}
	}
}
