# container-host

A tool for managing and running Fedora CoreOS virtual machines that work with Docker.

```shell
# go build 
make build
# executes the go binary
make run

DOCKER_HOST=tcp://localhost:2377 docker run -it hello-world:latest
```

You can download a release binary for your system in the Github releases.