# Charts 

### What is chart
Helm uses a packaging format called charts. Basically,  chart is a collection of files that describe a related set of K8s resources. A single chart might be used to deploy something simple,  or something complex, like a full web app with servers, databases, caches, and so on.
Charts are created as files laid out in a particular directory tree. They can be packaged into versioned archives to be deployed.

> Note: To work with IPAM Helm is required tool to be installed. 

### How to install 
If you have an access to the docker registry and k8s installation that you can use for development purposes, you may skip corresponding steps.
Otherwise use the link to find the corresponding steps for installation: 
[helm](https://helm.sh/) - the package manager for k8s that works with helm charts to define, install, and upgrade even the most complex application.

### Deployment

Operator can be deployed to the environment with kubectl, kustomize or Helm. User may choose one that is more suitable.

```sh
# install release "dev" to "onmetal" namespace
helm install dev ./chart/ -n onmetal --create-namespace
# remove release "dev" from "onmetal" namespace
helm uninstall dev -n onmetal
```

### Chart Folder Structure 
Chart folder  is organized as a collection of files inside of a directory. Inside of this directory, you will find certain information: 

- templates/    directory of templates that, when combined with values, will generate valid Kubernetes manifest files.
- .helmignore/   file is used to specify files you don't want to include in your helm chart.
- chart.yaml/  directory containing any charts upon which this chart depends.
- values.yaml/ values files can declare values for the top-level chart, as well as for any of the charts that are included in that chart's charts/ directory.














  
  