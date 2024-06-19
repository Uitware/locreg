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
)

var (
	location           = "eastus"
	resourceGroupName  = "sample-resource-group"
	appServicePlanName = "sample-appservice-plan"
	appServiceName     = "sample-appservice-app"
	dockregServerURL   = "https://index.docker.io/v1/"
	dockregUsername    = "username"
	dockregPassword    = "password"
	dockerImage        = "nginx"
	tag                = "latest"
)

var (
	resourcesClientFactory  *armresources.ClientFactory
	appserviceClientFactory *armappservice.ClientFactory
)

var (
	resourceGroupClient *armresources.ResourceGroupsClient
	plansClient         *armappservice.PlansClient
	webAppsClient       *armappservice.WebAppsClient
)

func Deploy() {
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

	resourceGroup, err := createResourceGroup(ctx)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Resource group created:", *resourceGroup.ID)

	appServicePlan, err := createAppServicePlan(ctx)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("App service plan created:", *appServicePlan.ID)

	appService, err := createWebApp(ctx, dockregServerURL, dockregUsername, dockregPassword, *appServicePlan.ID)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("App service created:", *appService.ID)
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
func createResourceGroup(ctx context.Context) (*armresources.ResourceGroup, error) {
	log.Println("Creating Resource Group...")
	resourceGroupResp, err := resourceGroupClient.CreateOrUpdate(
		ctx,
		resourceGroupName,
		armresources.ResourceGroup{
			Location: to.Ptr(location),
		},
		nil)
	if err != nil {
		return nil, err
	}
	return &resourceGroupResp.ResourceGroup, nil
}

func createAppServicePlan(ctx context.Context) (*armappservice.Plan, error) {
	log.Println("Creating App Service Plan...")
	pollerResp, err := plansClient.BeginCreateOrUpdate(
		ctx,
		resourceGroupName,
		appServicePlanName,
		armappservice.Plan{
			Location: to.Ptr(location),
			SKU: &armappservice.SKUDescription{
				Name:     to.Ptr("S1"),
				Capacity: to.Ptr[int32](1),
				Tier:     to.Ptr("Standard"),
			},
			Properties: &armappservice.PlanProperties{
				Reserved: to.Ptr(true),
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

func createWebApp(ctx context.Context, dockregServerURL, dockregUsername, dockregPassword, appServicePlanID string) (*armappservice.Site, error) {
	log.Println("Creating Web App...")

	pollerResp, err := webAppsClient.BeginCreateOrUpdate(
		ctx,
		resourceGroupName,
		appServiceName,
		armappservice.Site{
			Location: to.Ptr(location),
			Properties: &armappservice.SiteProperties{
				ServerFarmID: to.Ptr(appServicePlanID),
				SiteConfig: &armappservice.SiteConfig{
					AlwaysOn:       to.Ptr(true),
					LinuxFxVersion: to.Ptr(fmt.Sprintf("DOCKER|%s:%s", dockerImage, tag)),
					AppSettings: []*armappservice.NameValuePair{
						{
							Name:  to.Ptr("DOCKER_REGISTRY_SERVER_URL"),
							Value: to.Ptr(dockregServerURL),
						},
						{
							Name:  to.Ptr("DOCKER_REGISTRY_SERVER_USERNAME"),
							Value: to.Ptr(dockregUsername),
						},
						{
							Name:  to.Ptr("DOCKER_REGISTRY_SERVER_PASSWORD"),
							Value: to.Ptr(dockregPassword),
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
