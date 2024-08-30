## locreg deploy

`locreg deploy [provider]` command is used to deploy the infrastructure needed by your app into the cloud.

### Providers:
- `locreg deploy azure` - Azure Container Instances and Azure App Service container platforms are currently supported.
- `locreg deploy aws` -  AWS ECS platform is currently supported
- GCP container platforms coming soon

### Options
```
    -h, --help         help for push
    --env [path]       Path to the environment file.
```