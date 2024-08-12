package azure

import (
	"context"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerinstance/armcontainerinstance/v2"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Uitware/locreg/pkg/parser"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appservice/armappservice/v2"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
)

func getProjectRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		panic("Cant get current directory")
	}
	dir = filepath.Join(dir, "..", "..", "..")
	return dir
}

func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	var seededRand = rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

var ResourceGroup = "locreg-test-rg" + generateRandomString(5)
var AppServicePlanName = "locreg-test-plan" + generateRandomString(5)
var AppServiceName = "locreg-test-app" + generateRandomString(5)
var ContainerInstanceName = "locregtestaci" + generateRandomString(5)

func TestDeployAppService(t *testing.T) {
	// Load test configuration
	config, err := parser.LoadConfig(filepath.Join(getProjectRoot(), "test", "test_configs", "azure", "locreg.yaml"))
	config.Deploy.Provider.Azure.ResourceGroup = ResourceGroup
	config.Deploy.Provider.Azure.AppServicePlan.Name = AppServicePlanName
	config.Deploy.Provider.Azure.AppService.Name = AppServiceName

	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Setup context
	ctx := context.Background()

	// Authenticate using Azure credentials
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		t.Fatalf("Failed to authenticate: %v", err)
	}

	// Initialize Azure resource clients
	subscriptionID, err := getSubscriptionID()
	if err != nil {
		t.Fatalf("Failed to get subscription ID: %v", err)
	}

	resourcesClientFactory, err = armresources.NewClientFactory(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create resources client factory: %v", err)
	}

	appserviceClientFactory, err = armappservice.NewClientFactory(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create appservice client factory: %v", err)
	}

	resourceGroupClient = resourcesClientFactory.NewResourceGroupsClient()
	plansClient = appserviceClientFactory.NewPlansClient()
	webAppsClient = appserviceClientFactory.NewWebAppsClient()

	// Test: Create Resource Group
	t.Run("CreateResourceGroup", func(t *testing.T) {
		rg, err := createResourceGroup(ctx, config)
		if err != nil {
			t.Errorf("Failed to create resource group: %v", err)
		} else {
			log.Println("Resource Group ID:", *rg.ID)
		}
	})

	// Test: Create App Service Plan
	var appServicePlanID string
	t.Run("CreateAppServicePlan", func(t *testing.T) {
		appServicePlan, err := createAppServicePlan(ctx, config)
		if err != nil {
			t.Errorf("Failed to create app service plan: %v", err)
		} else {
			appServicePlanID = *appServicePlan.ID
			log.Println("App Service Plan ID:", appServicePlanID)
		}
	})
	envVars, _ := parser.LoadEnvVarsFromFile(filepath.Join(getProjectRoot(), "test", "test_configs", "azure", "env_example_locreg.env"))
	// Test: Create Web App
	t.Run("CreateWebApp", func(t *testing.T) {
		tunnelURL := "dummy-tunnel-url" // Replace with a valid tunnel URL or mock it for the test
		appService, err := createWebApp(ctx, config, appServicePlanID, tunnelURL, envVars)
		if err != nil {
			t.Errorf("Failed to create web app: %v", err)
		} else {
			log.Println("Web App ID:", *appService.ID)
		}
	})
}

func TestDeployContainerInstance(t *testing.T) {
	config, err := parser.LoadConfig(filepath.Join(getProjectRoot(), "test", "test_configs", "azure", "locreg_aci.yaml"))
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}
	config.Deploy.Provider.Azure.ResourceGroup = ResourceGroup
	config.Deploy.Provider.Azure.ContainerInstance.Name = ContainerInstanceName
	config.Registry.Username = os.Getenv("REGISTRY_USERNAME")
	config.Registry.Password = os.Getenv("REGISTRY_PASSWORD")

	// Setup context
	ctx := context.Background()

	// Authenticate using Azure credentials
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		t.Fatalf("Failed to authenticate: %v", err)
	}

	// Initialize Azure resource clients
	subscriptionID, err := getSubscriptionID()
	if err != nil {
		t.Fatalf("Failed to get subscription ID: %v", err)
	}

	aciClientFactory, err = armcontainerinstance.NewClientFactory(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create ACI client: %v", err)
	}

	resourceGroupClient = resourcesClientFactory.NewResourceGroupsClient()
	aciClient = aciClientFactory.NewContainerGroupsClient()
	envVars, _ := parser.LoadEnvVarsFromFile(filepath.Join(getProjectRoot(), "test", "test_configs", "azure", "env_example_locreg.env"))
	t.Run("DeployContainerInstance", func(t *testing.T) {
		aci, err := createACI(ctx, config, "docker.io", envVars)
		if err != nil {
			t.Fatalf("Failed to create ACI: %v", err)
		} else {
			log.Println("ACI created:", *aci.ID)
		}
	})
}

func TestCleanupResources(t *testing.T) {
	// Test cleanup function independently
	ctx := context.Background()

	// Initialize clients
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		t.Fatalf("Failed to authenticate: %v", err)
	}

	subscriptionID, err := getSubscriptionID()
	if err != nil {
		t.Fatalf("Failed to get subscription ID: %v", err)
	}

	resourcesClientFactory, err = armresources.NewClientFactory(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create resources client factory: %v", err)
	}

	appserviceClientFactory, err = armappservice.NewClientFactory(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create appservice client factory: %v", err)
	}

	aciClientFactory, err = armcontainerinstance.NewClientFactory(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create ACI client: %v", err)
	}

	resourceGroupClient = resourcesClientFactory.NewResourceGroupsClient()
	plansClient = appserviceClientFactory.NewPlansClient()
	webAppsClient = appserviceClientFactory.NewWebAppsClient()
	aciClient = aciClientFactory.NewContainerGroupsClient()

	tracker := &ResourceTracker{
		ResourceGroup:      ResourceGroup,
		AppServicePlan:     AppServicePlanName,
		WebApp:             AppServiceName,
		ContainterInstance: ContainerInstanceName,
	}
	cleanupResources(ctx, tracker)
}
