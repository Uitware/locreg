## locreg destroy

`locreg destroy [options]` command is used to destroy the resources created by locreg.

### Usage:
```bash
locreg destroy [option]
```
Destroy all the resources created by locreg and managed in `~/.locreg` profile file. If the `~/.locreg` profile file corrupted, resources won't be deleted.

### Options:
```
    -h, --help    help for destroy
    registry      Destroys the local container registry.
    tunnel        Destroys the public access tunnel.
    cloud         Destroys cloud resources (e.g., serverless instances).
    all           Destroys all resources, including registry, tunnel, and cloud resources.
```  
