## locreg push

`locreg push [location] [options]` is used to build and push the image to the local registry.

### Location
The directory where the **Dockerfile** is located. Must be specified and should be a valid path.
```bash
locreg push /path/to/Dockerfile
locreg push . # to build in the current directory
```


### Options
```
    -h, --help         help for push
    -t, --tag string   Tag of the image to be pushed. (defaults to "latest")
```