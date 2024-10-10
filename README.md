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
For locreg to work properly, you must have **Docker installed on your machine**
you can do this by following [Docker documentation](https://docs.docker.com/engine/install/).
Also, **Ngrok account must be created** you can do this in [Ngrok website](https://ngrok.com/).
Additionally, if you plan to use locreg with Azure, ensure that the **Azure CLI
is installed** and you have authenticated.
Here you can find how to install it [Azure CLI](https://docs.microsoft.com/en-us/cli/azure/install-azure-cli).

#### Go install

With Go 1.22+, build and install the latest released version:

```go install github.com/Uitware/locreg@latest```

#### Bash script

Use the following Bash script to install locreg from Github Releases:

```
curl -OL https://github.com/Uitware/locreg/releases/latest/download/locreg.tar.gz
tar -zxvf locreg.tar.gz
chmod +x locreg
sudo mv locreg /usr/local/bin/locreg

# remove tarball after installation: 
rm locreg.tar.gz
```

## ```locreg``` usage

Use ```locreg --help``` to display usage info.

## Base scenario - local registry
If you want to use `locreg` to create a local registry, and make it publicly available via Ngrok, you can use the following commands:
First prepare locreg.yaml file that is stored in same directory as your Docker file or where your project is located
with the following content:

```yaml
registry:
  port: 8080 # If omitted, default value 5000 will be assigned.
  username: "locreg" # If omitted, a randomly generated value will be assigned.
  password: "locreg" # If omitted, a randomly generated value will be assigned.

image:
  name: "sample-app" # Set your desired image name, if omitted, default value locreg-built-image will be assigned.
  tag: "latest"  # Set your desired image tag, if omitted, it will be set to your latest commit SHA, if no .git folder then it will be set to "latest".

tunnel:
  provider:
    ngrok:
      name: my-locreg-test # Set your desired name, if omitted, default value locreg-ngrok will be assigned.
      port: 5050 # Set your desired port number, if omitted, default value 4040 will be assigned.
      networkName: ngrok-network # Set your desired network name, if omitted, default value locreg-ngrok will be assigned.
```
> If you want better understanding of default values, then read - [locreg.yaml configuration options in docs](https://uitware.github.io/locreg/configuration/)

After that run command ``locreg registry`` - to start a local container registry and establish a ngrok tunnel for public access.
Now your registry is set up, and you can push your image to it with `locreg push .` command.

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
  port: 5555 # Port on which the local registry will run, if omitted, default value 5000 will be assigned.
  tag: "2" # Tag for the Docker registry image, if omitted, default value 2 will be assigned.
  image: "registry" # Docker image to use for the local registry, if omitted, default value registry will be assigned.
  name: "my-registry" # Name for the local registry container, if omitted, default value locreg-registry will be assigned.
  username: "myUsername" # Username for accessing the registry, if omitted, a randomly generated value will be assigned.
  password: "myPassword" # Password for accessing the registry, if omitted, a randomly generated value will be assigned.
```

🖼️ Image configuration
defines the name and tag for the Docker image that will be built and deployed.

```
image:
  name: "locreg-app" # Name of the Docker image to build and deploy, if omitted, default value locreg-built-image will be assigned.
  tag: "latest" # Tag for the Docker image (e.g., latest, v1.0.0), if omitted, it will be set to your latest commit SHA, if git is not initialised then it will be set to "latest".
```

🌐 Tunnel backend configuratio (Ngrok by default)

```
tunnel:
  provider:
    ngrok:
      name: "my-locreg-test" # Name for the Ngrok tunnel instance, if omitted, default value locreg-ngrok will be assigned.
      port: 5050 # Port on which the Ngrok tunnel will run, if omitted, default value 4040 will be assigned.
      networkName: "ngrok-network" # Name of the Docker network to which the Ngrok tunnel container will connect, if omitted, default value locreg-ngrok will be assigned.
```

> **_NOTE:_** that you should export ```NGROK_AUTHTOKEN``` in order to use Ngrok tunnel backend: 


☁️ Application backend (configuration of the serverless cloud runtime resource) example: 

☁️ Azure App Service:

```
deploy:
  provider:
    azure:
      location: "East US" # Azure location for the resources, if omitted, default value eastus will be assigned.
      resourceGroup: "LocregResourceGroup" # Name of the Azure resource group, if omitted, default value LocregResourceGroup will be assigned.
      appServicePlan:
        name: "LocregAppServicePlan" # Name of the App Service Plan, if omitted, default value LocregAppServicePlan will be assigned.
        sku:
          name: "B1" # Pricing tier (SKU) for the App Service Plan, if omitted, default value F1 will be assigned.
          capacity: 1 # Capacity of the plan (number of instances), if omitted, default value 1 will be assigned.
        planProperties:
          reserved: true # Indicates if the plan should use a reserved instance (for Linux), if omitted, default value true will be assigned.
      appService:
        name: "LocregAppService1112233" # Name of the App Service, if omitted, a randomly generated value will be assigned.
        siteConfig:
          alwaysOn: true # Keeps the app always running, if omitted, default value false will be assigned.
tags: # Tags for the cloud resources                   
  locreg-version: "0.1.0"
  test: "test"

# By default, locreg tags all resources as managed-by: locreg
# If you want to turn off the tags (which is not recommended), you can set it to `false` like in example below
#tags: false
```

☁️ Azure Container Instance:

```
deploy:
  provider:
    azure:
      location: "Poland Central" # Azure location for the resources, if omitted, default value eastus will be assigned.
      resourceGroup: "rg_locreg" # Name of the Azure resource group, if omitted, default value LocregResourceGroup will be assigned.
      containerInstance:
        name: "weatherappcontainer" # Name of the Container Instance, if omitted, default value locreg-container will be assigned.
        osType: "Linux" # Operating system type (e.g., Linux), if omitted, default value Linux will be assigned.
        restartPolicy: "OnFailure" # Restart policy for the container (e.g., Always, OnFailure), if omitted, default value Always will be assigned.
        ipAddress:
          type: "Public" # Type of IP address (e.g., Public, Private), if omitted, default value Public will be assigned.
          ports:
            - port: 80 # Port to expose, if omitted, default value 80 will be assigned.
              protocol: "TCP" # Protocol for the exposed port (e.g., TCP, UDP), if omitted, default value TCP will be assigned.
        resources:
          requests:
            cpu: 1.0 # Number of CPUs allocated, if omitted, default value 1.0 will be assigned.
            memory: 1.5 # Amount of memory allocated (in GB), if omitted, default value 1.5 will be assigned.

tags: # Tags for the cloud resources                   
  locreg-version: "0.1.0"
  test: "test"

# By default, locreg tags all resources as managed-by: locreg
# If you want to turn off the tags (which is not recommended), you can set it to `false` like in example below
#tags: false
```

> **_NOTE:_** You should authenticate with ```az``` CLI in order to use Azure application backend: https://learn.microsoft.com/en-us/cli/azure/reference-index?view=azure-cli-latest#az-login

☁️AWS ECS(Elastic Container Service):

```
deploy:
  provider:
    aws: # Specify the provider name
      region: "us-east-1" # AWS region where resources will be deployed. May be omitted
      ecs: # ECS service configuration
        clusterName: "myClusterName" # Name of the ECS cluster. May be omitted
        serviceName: "myServiceName" # Name of the ECS service. Must be unique. May be omitted
        serviceContainerCount: 1 # Number of containers to run. May be omitted
        taskDefinition:
          family: "myTaskFamily" # Name of the task family. May be omitted
          memoryAllocation: 512 # Memory allocated for the task in MB. May be omitted
          cpuAllocation: 256 # CPU units allocated for the task. May be omitted
          containerDefinitions:
            - name: "myContainerName" # Name of the container. Must be unique. May be omitted
              portMappings:
                - containerPort: 80 # Port number on the container. May be omitted
                  hostPort: 80 # Port number on the host. May be omitted
                  protocol: "tcp" # Protocol used by the container. May be omitted
      vpc: # VPC (Virtual Private Cloud) configuration
        cidrBlock: "10.0.0.0/16" # CIDR block for the VPC. May be omitted
        subnet:
          cidrBlock: "10.0.1.0/24" # CIDR block for the subnet. May be omitted
```

> **_NOTE:_** You should authenticate with ```aws``` CLI in order to use AWS application backend: https://docs.aws.amazon.com/cli/latest/userguide/getting-started-quickstart.html

## 📦 ```locreg``` Docs
Read detailed documentation on how to use ```locreg``` in here - [docs](https://uitware.github.io/locreg/)
