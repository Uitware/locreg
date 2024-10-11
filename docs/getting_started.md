# Get Started

### Setup 
[Install `locreg` and Prerequisites](./install.md)

>To start using `locreg` you need to have: `locreg`, `docker` and corresponding cloud's CLI installed on your machine.

### Import your tunnel credentials 
```bash
export NGROK_AUTHTOKEN=your_ngrok_auth_token
```
### Then authenticate with Azure 
```bash
az login
```

### Then authenticate with AWS
```bash
aws configure
```

### Copy configuration to file called `locreg.yaml` for Azure App Service deployment
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
This configuration creates a local registry, tunnel and deploys the image to the Azure App Service.

### Copy configuration to file called `locreg.yaml` for AWS ECS deployment
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
    aws:
      region: "us-east-1"
      ecs:
        clusterName: "locreg-cluster"
        serviceName: "locreg-service"
        serviceContainerCount: 1
        taskDefinition:
          family: "locreg-task"
          memoryAllocation: 512
          cpuAllocation: 256
          containerDefinitions:
            - name: "locreg-container"
              portMappings:
                - containerPort: 80
                  hostPort: 80
                  protocol: "tcp"
      vpc:
        cidrBlock: "10.10.0.0/16"
        subnet:
          cidrBlock: "10.10.10.0/24"

tags:
  managed-by: "locreg"
```
This configuration creates a local registry, tunnel and deploys the image to the AWS ECS.


### Create a sample Dockerfile
```Dockerfile
FROM nginx:alpine
RUN echo "Hello from locreg" > /usr/share/nginx/html/index.html
```

### Create registry, then build and push the image to the registry and deploy to Azure
```bash
locreg registry
locreg push
locreg deploy azure
```
> After you can go to the Azure portal and see the deployed app service.

### Create registry, then build and push the image to the registry and deploy to AWS
```bash
locreg registry
locreg push
locreg deploy aws
```

> After you can go to the AWS console and see the deployed ECS service.

---
## What's next?
- Use [getting started](./getting_started.md) guide to see how to use `locreg` to deploy your app.
- Get familiar with `locreg` using [locreg CLI](./cli/locreg.md) guide.