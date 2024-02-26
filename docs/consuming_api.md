# Consuming API with client

Package provides a client library written in go for programmatic interactions with API.

Clients for corresponding API versions are located in `clientset/` and act similar to [client-go](https://github.com/kubernetes/client-go).

Below there are two examples, for inbound (from the pod deployed on a cluster) and outbound (from the program running on 3rd party resources) interactions.

## Inbound example

```go
import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"

	apiv1alpha1 "github.com/ironcore-dev/ipam/api/v1alpha1"
	clientv1alpha1 "github.com/ironcore-dev/ipam/clientset"
)

func inbound() error {
	// get config from environment
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}

	// register CRD types in local client scheme
	if err := apiv1alpha1.AddToScheme(scheme.Scheme); err != nil {
		return errors.Wrap(err, "unable to add registered types to client scheme")
	}

	// create a client from obtained configuration
	cs, err := clientset.NewForConfig(config)
	if err != nil {
		return errors.Wrap(err, "unable to build clientset from config")
	}

	// get a client for particular namespace
	client := clientset.IpamV1Alpha1().Networks("default")

	// request a list of resources
	list, err := client.List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return errors.Wrap(err, "unable to get list of resources")
	}

	// print names of resources
	for _, r := range list.Items {
		fmt.Println(r.Name)
	}

	return nil
}
```

### Outbound example

```go
import (
    "context"
    "fmt"
    "path/filepath"
    
    "github.com/pkg/errors"
    "github.com/spf13/pflag"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/client-go/tools/clientcmd"
    "k8s.io/client-go/util/homedir"
    
    "k8s.io/client-go/kubernetes/scheme"

    apiv1alpha1 "github.com/ironcore-dev/ipam/api/v1alpha1"
    clientv1alpha1 "github.com/ironcore-dev/ipam/clientset"
)

func outbound() error {
	// make default path to kubeconfig empty
	var kubeconfigDefaultPath string

	// if there is a home directory in environment,
	// alter the defalut path to ~/.kube/config
	if home := homedir.HomeDir(); home != "" {
		kubeconfigDefaultPath = filepath.Join(home, ".kube", "config")
	}

	// configure the kubeconfig CLI flag with dafault value
	kubeconfig := pflag.StringP("kubeconfig", "k", kubeconfigDefaultPath, "path to kubeconfig")
	// configure k8s namespace flag with "default" default value
	namespace := pflag.StringP("namespace", "n", "default", "k8s namespace")
	// parse flags
	pflag.Parse()

	// read in kubeconfig file and build configuration
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		return errors.Wrapf(err, "unable to read kubeconfig from path %s", kubeconfig)
	}

	// register CRD types in local client scheme
	if err := apiv1alpha1.AddToScheme(scheme.Scheme); err != nil {
		return errors.Wrap(err, "unable to add registered types to client scheme")
	}

	// create a client from obtained configuration
	cs, err := clientset.NewForConfig(config)
        if err != nil {
        return errors.Wrap(err, "unable to build clientset from config")
    }

	// get a client for particular namespace
	client := clientset.IpamV1Alpha1().Networks(*namespace)
	
	// request a list of resources
	list, err := client.List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return errors.Wrap(err, "unable to get list of resources")
	}
	
	// print names of resources
	for _, r := range list.Items {
		fmt.Println(r.Name)
	}

	return nil
}
```

