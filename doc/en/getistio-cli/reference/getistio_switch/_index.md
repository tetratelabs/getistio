---
title: "getistio switch"
url: /getistio-cli/reference/getistio_switch/
---

Switch the active istioctl to a specified version

```
getistio switch [flags]
```

#### Examples

```
# Switch the active istioctl version to version=1.7.7, flavor=tetrate and flavor-version=0
$ getistio switch --version 1.7.7 --flavor tetrate --flavor-version=0, 

# Switch to version=1.8.3, flavor=istio and flavor-version=0 using name flag
$ getistio switch --name 1.8.3-istio-v0

# Switch from active version=1.8.3 to version 1.9.0 with the same flavor and flavor-version
$ getistio switch --version 1.9.0

# Switch from active "tetrate flavored" version to "istio flavored" version with the same version and flavor-version
$ getistio switch --flavor istio

# Switch from active version=1.8.3, flavor=istio and flavor-version=0 to version 1.9.0, flavor=tetrate and flavor-version=0
$ getistio switch --version 1.9.0 --flavor=tetrate

# Switch from active version=1.8.3, flavor=istio and flavor-version=0 to version=1.8.3, flavor=tetrate, flavor-version=1
$ getistio switch --flavor tetrate --flavor-version=1

# Switch from active version=1.8.3, flavor=istio and flavor-version=0 to the latest 1.9.x version, flavor=istio and flavor-version=0
$ getistio switch --version 1.9

```

#### Options

```
      --name string          Name of distribution, e.g. 1.9.0-istio-v0
      --version string       Version of istioctl, e.g. 1.7.4. When --name flag is set, this will not be used.
      --flavor string        Flavor of istioctl, e.g. "tetrate" or "tetratefips" or "istio". When --name flag is set, this will not be used.
      --flavor-version int   Version of the flavor, e.g. 1. When --name flag is set, this will not be used (default -1)
  -h, --help                 help for switch
```

#### Options inherited from parent commands

```
  -c, --kubeconfig string   Kubernetes configuration file
```

#### SEE ALSO

* [getistio](/getistio-cli/reference/getistio/)	 - GetIstio is an integration and lifecycle management CLI tool that ensures the use of supported and trusted versions of Istio.

