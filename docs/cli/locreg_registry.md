## locreg registry

`locreg registry [command] [options]` command is used to create a local registry and a tunnel to expose it to the public Internet.
Values for registry and tunnel are taken from the `locreg.yaml` configuration file.

### Commands:
- `locreg registry rotate` - Rotate the password of the registry.
- `locreg registry only-registry` - Create only the local registry, without exposing to the public Internet. #TODO in the next release

### Options:
```
    -h, --help    help for registry
```