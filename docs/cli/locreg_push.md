## locreg push

`locreg push [directory] [options]` is the command that you use to build and push the image to the local registry.

### Directory
The directory where the **Dockerfile** is located. Must be specified and should be a valid path.
```bash
locreg push /path/to/Dockerfile
locreg push . # For current directory
```


### Options
```
    -h, --help   help for push
    -t, --tag string   Tag of the image to be pushed. (default "latest")
```