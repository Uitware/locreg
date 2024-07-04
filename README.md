# 🚀☁️ ```locreg``` - streamline your cloud-native registryless development 

Serverless container runtimes like AWS ECS, Azure App Service for Containers, GCP Cloud Run, etc. are extremely popular nowadays, but deployment of these resources always require to spin up a container runtime resource, as well a separate cloud-based container registry, manually, via proprietary CLI tool, or using IaC tools. 

```locreg``` enables **registryless** approach for serverless applications deployment - you need just a single simple configuration file and locreg binary installed. It: 

- 📍 spins up a **local** container registry
- 🛠️ **builds** the container image and **pushes** to local registry
- 🌐 spins up a **temporary tunnel** to expose local registry to the Internet (Ngrok is supported for now, Cloudflared coming soon)
- 🚀 deploys a serverless container runtime resource (Azure App Service supported for now, AWS ECS and GCP Cloud Run coming soon)
- 🔑 passes the credentials of publicly exposed local registry to serverless container runtime resource to streamline the deployment 

 **Your application is up and running on a cloud serverless platform! 🎉** Now you can easily **rebuild** and **redeploy** your application using ```locreg```, and **when the testing is done - easily destroy both local and cloud resources via ```locreg```**!

All configuration is defined and controlled via single ```locreg.yaml``` file - please see ```config_example.yaml``` in the repo root. 

#### 📄 ```locreg``` concepts

locreg uses ```locreg.yaml``` as a source of truth for development environment that it creates. Configuration should include a single registry backend, a single application backend and a single tunnel backend.

🗃️ Registry backend (reference ```distribution``` registry is used by default: https://distribution.github.io/distribution/)

```
registry:
  port: 5555
  tag: "2"
  image: "registry"
  name: "my-registry"
  username: "myUsername"
  password: "myPassword"
```

☁️ Application backend (configuration of the serverless cloud runtime resource) example: 

```
deploy:
  provider:
    azure:
      location: "East US" # Azure location for resources
      resourceGroup: "myResourceGroup" # RG name
      appServicePlan:
        name: "myAppServicePlan" # App Service Plan name
        sku:
          name: "F1" # ASP SKU
          capacity: 1 # ASP capacity
          tier: "FREE" # ASP tier
        planProperties:
          reserved: true # ASP reserved option
      appService:
        name: "myAppService" # App Service name
        siteConfig:
          alwaysOn: true # AS always on option
          dockerRegistryServerUrl: "https://index.docker.io/v1/"
          dockerImage: "myDockerUsername/myImage"
          tag: "latest"
```

Note that you should authenticate with ```az``` CLI in order to use Azure application backend: https://learn.microsoft.com/en-us/cli/azure/reference-index?view=azure-cli-latest#az-login


🌐 Tunnel backend configuratio (Ngrok by default)

```
tunnel:
  provider:
    ngrok
```

Note that you should export ```NGROK_AUTHTOKEN``` in order to use Ngrok tunnel backend: 


## ```locreg``` installation

Currently supported platforms include only Linux amd64. 
We're planning to add MacOS ARM and Linux ARM support soon.
There are several ways to install locreg:

#### Go install

With Go 1.16+, build and install the latest released version:

```go install github.com/Uitware/locreg@latest```

#### Bash script

Use the following Bash script to install locreg from Github Releases:

```
curl -OL https://github.com/Uitware/locreg/releases/download/v0.1.1-alpha/locreg.tar.gz
tar -zxvf locreg.tar.gz
chmod +x locreg
mv locreg /usr/local/bin/locreg

# to clean resources: 
rm locreg.tar.gz
```

## ```locreg``` usage

Use ```locreg --help``` to display usage info.

Commands: 

- ```deploy``` - create a cloud provider's serverless container runtime resource and deploy your application
- ```push``` - build and push a container image to the local registry
- ```registry``` - start a local container registry
- ```tunnel``` - spin up a tunnel to expose local container registry to the public Internet
- ```env``` - manage the environment variables used by locreg