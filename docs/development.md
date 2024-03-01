# Development

## Running locally without deployment 

It is possible to run operator locally during development. To do this, execute `make run`. In that case 
operator will use current context from kubeconfig to connect to k8s' apiserver.

If problem running webhook occures, use `ENABLE_WEBHOOKS=false` env var to disable them on local run.

## Adding a new API version

One should not modify API once it got to be used.

Instead, in order to introduce breaking changes to the API, a new API version should be created.

First, move the existing controller to a different file, as generator will try to put a new controller into the same location, e.g.

    mv controllers/network_controller.go controllers/network_v1alpha1_controller.go

After that, add a new API version

    operator-sdk create api --group machine --version v1alpha2 --kind Network --resource --controller

Do modifications in a new CR, add a new controller to `main.go`.

Following actions should be applied to other parts of project:
- regenerate code and configs with `make install`
- add a client to client set for the new API version
- alter Helm chart with new CRD spec

## Deprecating old APIs

Since there is no version deprecation marker available now, old APIs may be deprecated with `kustomize` patches

Describe deprecation status and deprecation warning in patch file, e.g. `crd_patch.yaml`

```
- op: add
  path: "/spec/versions/0/deprecated"
  value: true
- op: add
  path: "/spec/versions/0/deprecationWarning"
  value: "This API version is deprecated. Check documentation for migration instructions."
```

Add patch instructions to `kustomization.yaml`

```
patchesJson6902:
  - target:
      version: v1
      group: apiextensions.k8s.io
      kind: CustomResourceDefinition
      name: ipam.ironcore.dev
    path: crd_patch.yaml
```

When you are ready to drop the support for the API version, give CRD a `+kubebuilder:skipversion` marker,
or just remove it completely from the code base.

This includes:
- API objects
- client
- controller
