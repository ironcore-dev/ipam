# IPAM
k8s operator for Ipam

### CRD parameters

| Parameter  | Description | Example | Validation rules |
| ------------- | ------------- | ------------- | ------------- |
| subnet | Subnet reference | subnet | Should exist |
| crd | Any other CRD to link IP request for | groupVersion, kind and name for CRD too lookup | Optional, should exist if specified |
| ip | IP to request | 10.12.34.64 | Optional, if not specified it will be assigned automatically in the specified subnet if any IPs are available |

## Getting started

This repo references other CRDs and you need to install them to proceed:
- Subnet 

### Required tools

Following tools are required to make changes on that package.

- k8s cluster access to deploy and test the result (via minikube or docker desktop locally)
- [make](https://www.gnu.org/software/make/) - to execute build goals
- [golang](https://golang.org/) - to compile the code
- [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/) - to interact with k8s cluster via CLI
- [kustomize](https://kustomize.io/) - to generate deployment configs
- [kubebuilder](https://book.kubebuilder.io) - framework to build operators
- [operator framework](https://operatorframework.io/) - framework to maintain project structure
- [helm](https://helm.sh/) - to work with helm charts

### Install definitions

In order to build and deploy, execute following command set: `make install`

### Development

So you might need to run `go env -w GOPRIVATE='github.com/onmetal/*'` first to build it.

This CRD is using webhooks so it can't be run in normal manner until webhooks are disabled.
To do so `ENABLE_WEBHOOKS=false` environment variable could be set.

So to run controller for development without deploy do: `make run ENABLE_WEBHOOKS=false`

### Deploy 

Docker registry is required to build and push an image. 
For local development you can use local registry e.g. `localhost:5000` for [docker desktop](https://docs.docker.com/registry/deploying/).

Replace with your registry if you're using quay or anything else.

```
# ! Be sure to install CRDs first
# Build and push Docker image
make docker-build docker-push IMG="localhost:5000/onmetal/ipam:latest" GIT_USER=yourusername GIT_PASSWORD=youraccesstoken

# Deploy
make deploy IMG="localhost:5000/onmetal/ipam:latest"
```

### Helm chart

```
# Deploy
helm install ipam ./chart/ -n ipam --create-namespace
# Undeploy
helm uninstall ipam -n ipam
```

### Use

`./config/samples/` directory contains examples of manifests. They can be used to try out the controller.

```
# Create subnets
# Create IPAM
kubectl apply -f config/samples/ipam_v1alpha1_ipam.yaml
# Check that IP was assigned -> should be 10.12.34.64
kubectl describe ipams ipam1
```

### Cleanup

`make undeploy`

### Testing

```
# Go to webhook directory
cd api/v1alpha1

# Run tests
go test . -v -ginkgo.v
```
