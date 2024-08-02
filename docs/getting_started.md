# Get Started

### Setup 
[Install `locreg` and Prerequisites](./install.md)

>To start using `locreg` you need to have: `locreg`, docker and Azure installed on your machine.

### Import your tunnel credentials 
```bash
exec NGROK_AUTHTOKEN=your_ngrok_auth_token
```
### Then Authenticate with Azure 
```bash
az login
```

### Copy configuration to file called `locreg.yaml`
```yaml
registry:
  port: 8080
  username: "locreg"

image:
  name: "sample-app"
  tag: "latest"

tunnel:
  provider:
    ngrok:
      name: my-locreg-test
      port: 5050
      networkName: ngrok-network

deploy:
  provider:
    azure:
      location: "East US"
      resourceGroup: "LocregGettingStarted"
      appServicePlan:
        sku:
          name: "F1"
          capacity: 1
        planProperties:
          reserved: true
      appService:
        siteConfig:
          alwaysOn: false

tags:
  managed-by: "locreg"
```
This configuration creates a local registry, tunnel and deploys the image to Azure App Service.

### Crate a sample Dockerfile
```Dockerfile
FROM nginx:alpine
RUN echo "Hello from locreg" > /usr/share/nginx/html/index.html
```

### Create registry then build and push the image
```bash
locreg registry
locreg push
locreg deploy azure
```
> After you can go to the Azure portal and see the deployed app service.

---
## What's next?
- Get familiar with `locreg` [configuration options](./configuration.md)
- Now take a look at the [locreg cli](./cli/locreg.md) to see how to use the `locreg` command line interface.
