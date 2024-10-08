# Installation 

Currently supported platforms include only Linux amd64. We're planning to add macOS ARM and Linux ARM support soon.
You may also install and use it on Windows machine, but first you would need to install WSL. There are several ways to install locreg:

# Change to public repo
### Using the go install command 
```bash
go install github.com/Uitware/locreg@latest  
```

### Using a bash script
```bash
curl -OL https://github.com/Uitware/locreg/releases/latest/download/locreg.tar.gz
tar -zxvf locreg.tar.gz
chmod +x locreg
mv locreg /usr/local/bin/locreg

# to clean resources: 
rm locreg.tar.gz
```

---
# Prerequisites 
### docker
You should also have docker installed on your machine. If you don't have a docker installed, you can install it by 
following the instructions [here](https://docs.docker.com/get-docker/).

## Azure CLI
If you are planning to deploy your images to Azure, you should also have Azure CLI installed on your machine. 
You can install it by following the instructions [here](https://docs.microsoft.com/en-us/cli/azure/install-azure-cli).

---
## What's next?
- Get familiar with `locreg` [configuration options](./configuration.md)
- Get familiar with [locreg](./cli/locreg.md) to see how to use the `locreg` command line interface.
- Get started with [getting started](./getting_started.md) guide to see how to use `locreg` to deploy your app.
