# Configuration file
`locreg` uses a configuration file to store the settings. The configuration file is a `.yaml` file with name `locreg.yaml`. 
The configuration file is stored in the same folder as Dockerfile or at the root of the project.


## Configuration file structure
The locreg configuration file consists of five sections as shown below.
```yaml
registry:

image:

tunnel:
  provider:


deploy:
  provider:

tags:
```
Each of the parts corresponds to a single item that `locreg` creates except of tags. 
The `tags` part is used to store the tags that are used to tag cloud resources


## The default values
Before going further to configuration, **it's important to understand how default values work.** 
If you want to use defaults, you must specify a type of service for which you want to use defaults. Like in this example below:
```yaml
deploy:
  provider:
    azure:
    location: "eastus"
    resourceGroup: "LocregResourceGroup"
    containerInstance:
```
The `containerInstance:` in this configuration has default values for all the properties. If you want to set some particular properties to non-default values, you can do it like in the example below:
```yaml
deploy:
  provider:
    azure:
    location: "eastus"
    resourceGroup: "LocregResourceGroup"
    containerInstance:
      name: "Sample-Conainerinstance"
      restartPolicy: "OnFailure"
```
> This way, all the values which you haven't explicitly overridden, are set to the defaults.

## Registry configuration
The registry configuration part is used to store the settings of the local registry. The registry configuration part consists of the following items.
```yaml
registry:
  port: 5555 # Port number of the registry may be omitted
  tag: "2" # Tag of the registry may be omitted
  image: "registry" # Image of the registry may be omitted
  name: "my-registry" # Name of the registry may be omitted
  username: "myUsername" # Username of the registry may be omitted
  password: "myPassword" # Password of the registry may be omitted
```
> As you can see, all configuration items for registry are optional. So if you want you can only specify `registry:` in your config, and it will be launched with all the default values.

### Registry default values
The default values for the registry configuration are predefined by the `locreg`, except of password and username witch are randomly generated 32 characters long strings.
Default values are as follows:
```yaml
registry:
  port: 5000
  tag: "2"
  image: "registry"
  name: "locreg-registry"
  password: cd322517461e36a0d08a38a6bbca66ffb774fe381555278a70ec56bb993a8ee1 #randomly generated 32 characters long string
  username: 866a7f4c2e38bbfbb67a5c487bd43d7e5773ed176b11987afbfbd2c7114090219dd26c88 #randomly generated 32 characters long string
```


## Image configuration
The image configuration part is used to store the settings of the image that is used to deploy the registry. The image configuration part consists of the following items.
```yaml
image:
  name: "your desired name" # Name of the image may be omitted
  tag: "version" # Tag of the image may be omitted
```

### Image default values
By default, the image configuration is set to the following values:
```yaml
image:
  name: "locreg-built-image"
  tag: # your current git SHA or "latest", if git repo isn't initialized 
```

## Tunnel configuration 
Tunnel configuration part is used to store the settings of the tunnel provider. The tunnel configuration part consists of the following items.
```yaml
tunnel:
  provider: # specify the provider name
    ngrok: # provider name
      name: "your ngrok container name" # Ngrok container name may be omitted
      image: "ngrok/ngrok" # Ngrok image may be omitted
      tag: "latest" # Ngrok image tag may be omitted
      port: 4040 # Ngrok port may be omitted
      networkName: "your network name" # Ngrok network may be omitted
```

### Tunnel default values
By default, the tunnel configuration is set to the following values:
```yaml
tunnel:
  provider: 
    ngrok: 
      name: locreg-ngrok
      image: ngrok/ngrok
      tag: latest
      port: 4040
      networkName: locreg-ngrok
```

## Deploy configuration
Deploy configuration part is used to store the settings of the deployment provider. The deployment configuration part consists of the following items:
```yaml
deploy:
  provider:
    azure: # specify the provider name
      appServicePlan: # App service plan configuration
      appService: # App service configuration
      # Or
      containerInstance: # Container instance configuration
```
It's important that only one of the two services can be specified in the deployment configuration. 
It's either `appService` and `appServicePlan` or `containerInstance`.
### Deployment for Azure App Service:
```yaml

deploy:
  provider:
    azure:
      location: "East US" # Location of all resources. May be omitted
      resourceGroup: "myResourceGroup" # Resource group name. May be omitted
      appServicePlan:
        name: "myAppServicePlan" # App service plan name. May be omitted
        sku:
          name: "F1" # App service plan SKU name. May be omitted
          capacity: 1 # App service plan capacity. May be omitted
        planProperties:
          reserved: true # App service plan reserved, may be omitted
      appService:
        name: "myAppService" # App service name MUST be unique. May be omitted
        siteConfig:
          alwaysOn: true # App service always on. May be omitted
```

#### App Service default values
By default, the deployment configuration is set to the following values:
```yaml
deploy:
  provider:
    azure:
      location: "eastus"
      resourceGroup: "LocregResourceGroup"
      appServicePlan:
        name: "LocregAppServicePlan"
        sku:
          name: "F1"
          capacity: 1
        planProperties:
          reserved: true
      appService:
        name: "locregappservice[random_suffix]" # [random_suffix] is a randomly generated 8 characters long string
        siteConfig:
          alwaysOn: false
```

### Deployment for Container Instances:
```yaml
deploy:
  provider:
    azure:
      location: "Poland Central" # Location of all resources. May be omitted
      resourceGroup: "rg_locreg" # Resource group name. May be omitted
      containerInstance: # Container instance configuration
        name: "weatherappcontainer" # Container instance name. May be omitted
        osType: "Linux" # Container instance OS type. May be omitted
        restartPolicy: "OnFailure" # Container instance restart policy. May be omitted
        ipAddress:
          type: "Public" # Container instance IP address type. May be omitted
          ports:
            - port: 80 # Container instance port. May be omitted
              protocol: "TCP" # Container instance protocol. May be omitted
        resources:
          requests: # Container instance resource requests. May be omitted
            cpu: 1.0 # Container CPU. May be omitted
            memory: 1.5 # Container memory. May be omitted
```

#### Container Instances default values
By default, the deployment configuration is set to the following values:
```yaml
deploy:
  provider:
    azure:
      location: "eastus"
      resourceGroup: "LocregResourceGroup"
      containerInstance:
        name: "locreg-container"
        osType: "Linux"
        restartPolicy: "Always"
        ipAddress:
          type: "Public"
          ports:
            - port: 80
              protocol: "TCP"
        resources:
          requests:
            cpu: 1.0
            memory: 1.5
```

## Tags configuration
Tags configuration part is used to store the tags that are used to tag cloud resources
The tags configuration part consists of the following items:
```yaml
tags:
  managed-by: "locreg" # May be omitted
```

### Multiple tags
Also, you can specify multiple tags like this:
```yaml
tags:
  managed-by: "locreg"
  environment: "development"
  owner: "John Doe"
  application: "weather-app"
  version: "1.0.0"
```

### Tags turning on/off
Tags part can be omitted, but if you do tag `managed-by: locreg` will be still added to all resources created by `locreg`.
To disable tags, you can explicitly set `tags:` to `false` as shown bellow:  
**Notice that it isn't recommended to disable tags - they allow to easily identify locreg-managed resources**
```yaml
tags: false
```
---
## Configuration file for env variables
To specify environment variables in the configuration file, you should use a .env file. The .env file should be placed in 
the same folder as the configuration file or in the repo root. The .env file should contain the environment variables in the following format:
```env
APP_NAME=MyApp
APP_ENV=development
APP_PORT=8080
DATABASE_URL=postgres://user:password@localhost:5432/myappdb
```
And to apply it, you should use command `locreg deploy azure --env`, which deploys your app with the specified environment variables.

---
## What's next?
- Use [getting started](./getting_started.md) guide to see how to use `locreg` to deploy your app.
- Get familiar with `locreg` using [locreg CLI](./cli/locreg.md) guide.