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


[![GitHub Pages](https://img.shields.io/badge/docs-GitHub%20Pages-blue.svg)](https://uitware.github.io/locreg/)
[![Godoc](https://pkg.go.dev/badge/github.com/Uitware/locreg?utm_source=godoc)](https://pkg.go.dev/github.com/Uitware/locreg)
[![Go Report Card](https://goreportcard.com/badge/github.com/Uitware/locreg)](https://goreportcard.com/report/github.com/Uitware/locreg)
[![Tests on Linux](https://github.com/Uitware/locreg/actions/workflows/binary-build.yml/badge.svg)](https://github.com/Uitware/locreg/actions/workflows/binary-build.yml)

## ```locreg``` installation

Currently supported platforms include only Linux amd64. 
We're planning to add MacOS ARM and Linux ARM support soon.
There are several ways to install locreg:


#### Prerequisites
For locreg to work properly, you must have Docker installed on your machine.  
Additionally, if you plan to use locreg with Azure, ensure that the Azure CLI
is installed and authenticated

#### Go install

With Go 1.22+, build and install the latest released version:

```go install github.com/Uitware/locreg@latest```

#### Bash script

Use the following Bash script to install locreg from Github Releases:

```
curl -OL https://github.com/Uitware/locreg/releases/download/latest/locreg.tar.gz
tar -zxvf locreg.tar.gz
chmod +x locreg
sudo mv locreg /usr/local/bin/locreg

# remove tarball after installation: 
rm locreg.tar.gz
```

## ```locreg``` usage

Use ```locreg --help``` to display usage info.

Commands: 

- ```deploy``` - creates a serverless container runtime resource with a specified cloud provider and deploys your application. Use this command with a provider (e.g., `azure`) and optionally specify an environment file using --env and the path to a .env file.
```
locreg deploy azure --env path/to/envfile
```

- ```push``` - build from the specified directory that contains Dockerfile for your container image and push the image to your local registry
```
locreg push path/to/dockerfile
```
- ```registry``` - start a local container registry
```
locreg registry
```
- ```tunnel``` - spin up a tunnel to expose local container registry to the public Internet
```
locreg tunnel
```
- ```destroy``` - removes resources created by locreg
  - `registry`: Destroys the local container registry.
  - `tunnel`: Destroys the public access tunnel.
  - `cloud`: Destroys cloud resources (e.g., serverless instances).
  - `all`: Destroys all resources, including registry, tunnel, and cloud resources.
```  
locreg destroy all
locreg destroy registry
  ```
## 📄 ```locreg``` concepts

locreg uses ```locreg.yaml``` as a source of truth for development environment that it creates. Configuration should include a single registry backend, a single application backend and a single tunnel backend.

🗃️ Registry backend (reference ```distribution``` registry is used by default: https://distribution.github.io/distribution/)

```
registry:
  port: 5555              # Port on which the local registry will run
  tag: "2"                # Tag for the Docker registry image
  image: "registry"       # Docker image to use for the local registry
  name: "my-registry"     # Name for the local registry container
  username: "myUsername"  # Username for accessing the registry
  password: "myPassword"  # Password for accessing the registry
```

🖼️ Image configuration
defines the name and tag for the Docker image that will be built and deployed.

```
image:
  name: "locreg-app"   # Name of the Docker image to build and deploy
  tag: "latest"        # Tag for the Docker image (e.g., latest, v1.0.0)
```

🌐 Tunnel backend configuratio (Ngrok by default)

```
tunnel:
  provider:
    ngrok:
      name: "my-locreg-test"        # Name for the Ngrok tunnel instance
      port: 5050                    # Port on which the Ngrok tunnel will run
      networkName: "ngrok-network"  # Name of the Docker network to which the Ngrok tunnel container will connect

```

Note that you should export ```NGROK_AUTHTOKEN``` in order to use Ngrok tunnel backend: 


☁️ Application backend (configuration of the serverless cloud runtime resource) example: 

☁️ Azure App Service:

```
deploy:
  provider:
    azure:
      location: "East US"                       # Azure location for the resources
      resourceGroup: "LocregResourceGroup"      # Name of the Azure resource group
      appServicePlan:
        name: "LocregAppServicePlan"            # Name of the App Service Plan
        sku:
          name: "B1"                            # Pricing tier (SKU) for the App Service Plan
          capacity: 1                           # Capacity of the plan (number of instances)
        planProperties:
          reserved: true                        # Indicates if the plan should use a reserved instance (for Linux)
      appService:
        name: "LocregAppService1112233"         # Name of the App Service
        siteConfig:
          alwaysOn: true                        # Keeps the app always running
tags:                                           # Tags for the cloud resources                   
  locreg-version: "0.1.0"
  test: "test"

# By default, locreg tags all resources as managed-by: locreg
# If you want to turn off the tags (which is not recommended), you can set it to `false`` like in example below
#tags: false


```

☁️ Azure Container Instance:

```
deploy:
  provider:
    azure:
      location: "Poland Central"       # Azure location for the resources
      resourceGroup: "rg_locreg"       # Name of the Azure resource group
      containerInstance:
        name: "weatherappcontainer"    # Name of the Container Instance
        osType: "Linux"                # Operating system type (e.g., Linux)
        restartPolicy: "OnFailure"     # Restart policy for the container (e.g., Always, OnFailure)
        ipAddress:
          type: "Public"               # Type of IP address (e.g., Public, Private)
          ports:
            - port: 80                 # Port to expose
              protocol: "TCP"          # Protocol for the exposed port (e.g., TCP, UDP)
        resources:
          requests:
            cpu: 1.0                   # Number of CPUs allocated
            memory: 1.5                # Amount of memory allocated (in GB)

tags:                                  # Tags for the cloud resources                   
  locreg-version: "0.1.0"
  test: "test"

# By default, locreg tags all resources as managed-by: locreg
# If you want to turn off the tags (which is not recommended), you can set it to `false`` like in example below
#tags: false
```

Note that you should authenticate with ```az``` CLI in order to use Azure application backend: https://learn.microsoft.com/en-us/cli/azure/reference-index?view=azure-cli-latest#az-login

## 📦 ```locreg``` Docs
Read detailed documentation on how to use ```locreg``` in here - [docs](https://uitware.github.io/locreg/)
