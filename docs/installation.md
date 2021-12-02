# Installation and Deployment

## Getting started

### Required tools

Following tools are required to work on that package.

- k8s cluster access to deploy and test the result (via minikube or docker desktop locally)
- [make](https://www.gnu.org/software/make/) - to execute build goals
- [golang](https://golang.org/) - to compile the code
- [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/) - to interact with k8s cluster via CLI
- [kustomize](https://kustomize.io/) - to generate deployment configs
- [kubebuilder](https://book.kubebuilder.io) - framework to build operators
- [operator framework](https://operatorframework.io/) - framework to maintain project structure
- [helm](https://helm.sh/) - to work with helm charts
- [cert-manager](https://cert-manager.io/) - to issue certificates for webhook endpoints

## Prepare environment

If you have an access to the docker registry and k8s installation that you can use for development purposes, you may skip
corresponding steps.

Otherwise, create a local instance of docker registry and k8s.

    # start minikube
    minikube start
    # enable registry
    minikube addons enable registry
    # run proxy to registry
    docker run --rm -d --name registry-bridge --network=host alpine ash -c "apk add socat && socat TCP-LISTEN:5000,reuseaddr,fork TCP:$(minikube ip):5000"
    # install cert-manager for k8s cluster
    # check "Releases" page to fetch up-to-date version
    # https://github.com/jetstack/cert-manager/releases 
    kubectl apply -f https://github.com/jetstack/cert-manager/releases/download/v1.1.1/cert-manager.yaml

## Build and install

In order to build and deploy, execute following command set.

Docker registry is required to build and push an image.

Replace `localhost:5000` with your registry if you're using quay or anything else.

    # generate code and configs
    make fmt vet generate manifests kustomize
    # build container and push to local registry
    make docker-build docker-push IMG="localhost:5000/onmetal/ipam:latest"
    # deploy controller
    make deploy IMG="localhost:5000/onmetal/ipam:latest"

Check `Makefile` for the full list of `make` goals with descriptions.

### Use

`./config/samples/` directory contains examples of manifests. They can be used to try out the controller.

    # apply config
    kubectl apply -f ./config/samples/ipam_v1alpha1_network.yaml
    # get resources
    kubectl get networks
    # get sample resource
    kubectl describe network network-sample

### Clean

After development is done, clean up local environment.

    # generate deployment config and delete corresponding entities
    kustomize build config/default | kubectl delete -f -
    # remove registry bridge
    docker stop registry-bridge
    # stop minikube
    minikube stop

## Deployment

Operator can be deployed to the environment with kubectl, kustomize or Helm. User may choose one that is more suitable.

### With kubectl using kustomize configs

    # deploy
    kubectl apply -k config/default/
    # remove
    kubectl delete -k config/default/

### With kustomize

    # build and apply
    kustomize build config/default | kubectl apply -f -
    # build and remove
    kustomize build config/default | kubectl delete -f -

### With Helm chart

    # install release "dev" to "onmetal" namespace
    helm install dev ./chart/ -n onmetal --create-namespace
    # remove release "dev" from "onmetal" namespace
    helm uninstall dev -n onmetal