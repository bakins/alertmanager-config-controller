alertmanager-config-controller
==============================

Kubernetes controller that generate a ConfigMap containing a [Prometheus Alertmanager](https://github.com/prometheus/alertmanager) configuration.

As Alertmanager does not allow include files for its configuration, this will collect
the data from other ConfigMaps and generate an Alertmanager config.

Heavily influenced by [konfd](https://github.com/kelseyhightower/konfd)

Usage
=====

```
$ ./alertmanager-config-controller
Collects alertmanager configs as defined in configmaps and generates a single config

Usage:
  alertmanager-config-controller [target-namespace] [target-name] [flags]

Flags:
  -e, --endpoint string          kubernetes endpoint (default "http://127.0.0.1:8001")
  -n, --namespace stringArray    namespace to query. can be used multiple times. default is all namespaces
  -o, --onetime                  run one time and exit.
  -s, --selector string          label selector
  -i, --sync-interval duration   the time duration between processing. (default 1m0s)
```  

> The controller assumes you are running `kubectl` in proxy mode to handle authentication with
Kubernetes.

> You may want to use [configmap-reload](https://github.com/jimmidyson/configmap-reload)
to reload Alertmanager when the configmap changes.  An HTTP POST to `/-/reload`
in Alertmanager will tell it to reload its config.  Alertmanager logs will show any errors.

The controller will list all ConfigMaps - optionally using a [label selector](https://kubernetes.io/docs/user-guide/labels/) and/or limiting to certain namespaces.

> ConfigMaps were used rather than [Third Party Resources](https://kubernetes.io/docs/user-guide/thirdpartyresources/) as various tools (helm, kubectl, etc have issues with TPRs.
One some of those issues are resolved, the controller my use TPRs.

The controller supports all the current configuration options for Alertmanager. It expects
certain annotations to be set on the ConfigMaps.  Only a single "object" may be defined in a config map.
All ConfigMaps expect to have the Alertmanager data in the `spec` key of the config map.

The controller will hash the existing ConfigMap data - if it exists - and the generated data
and only update if these do not match.

The controller will always skip the target ConfigMap when gathering config maps for consideration
of config snippets.

## Alertmanager Configuration "Types"

The "type" of the ConfigMap is set using the `alertmanager-type` annotation key. The value of the key is case-insenstive as it is lowercased before being checked.

An invalid type is ignored. Any syntax error within the ConfigMap data `spec` will cause the controller to
error and it will not generate a config map.

### Global

A "global" type specifies the configuration for the [global](https://prometheus.io/docs/alerting/configuration/) section of the Alertmanager configuration.  Currently, the controller does not check for duplicates and the section will simple be overwritten.

Example:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: global-alertmanager-config
  namespace: kube-system
  labels:
    type: alertmanager
    kubernetes.io/cluster-service: "true"
  annotations:
    alertmanager-type: Global
data:
  spec: |-
    resolve_timeout: 60s
    smtp_smarthost: 'localhost:25'
    smtp_from: 'alertmanager@example.org'
```

In this example, we expect the controller to be ran with the selector argument set like `--sector=type=alertmanager`.

The controller will parse the `spec` to ensure it is a valid global configuration.

### InhibitRule

InhibitRule generates a single [inhibit_rule](https://prometheus.io/docs/alerting/configuration/#inhibit-rule-<inhibit_rule>). All the rules are added to a list and this generates the `inhibit_rules` section of the Alertmanager config.

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: my-inhibit-rule
  namespace: kube-system
  labels:
    type: alertmanager
  annotations:
    alertmanager-type: InhibitRule
data:
  spec: |-
    target_match:
      foo: bar
```

### Receiver

Receiver generates a single [receiver](https://prometheus.io/docs/alerting/configuration/#receiver-<receiver>). All the receivers are added to a list and set as the `receivers` section.  See the Alertmanager documents for the format
of each receiver type.

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: alertmanager-default-webhook
  namespace: kube-system
  labels:
    type: alertmanager
    kubernetes.io/cluster-service: "true"
  annotations:
    alertmanager-type: Receiver
data:
  spec: |-
    name: mywebhook

    webhook_configs:
     - url: http://alertmanager-webhook/alerts
       send_resolved: true
```

### Template

Template expects a single string that is a path for loading templates.   Templates are not handled
by the controller currently.  A second ConfigMap - possible generated using the [configmap-aggregator](https://github.com/bakins/configmap-aggregator) - could be used to store
templates.  Only the path, as seen by the Alertmanager process, needs to be set here.

### Route

Route generates a single [route](https://prometheus.io/docs/alerting/configuration/#route-<route>).
one - and only one - route should be marked as the default route by setting the `alertmanager-default-route: "true"` annotation. Specifying zero or more than one will cause the controller to error.
The other routes are added as the list of `routes` on this route. Any nested routes are ignored.

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: alertmanager-default-route
  namespace: kube-system
  labels:
    type: alertmanager
    kubernetes.io/cluster-service: "true"
  annotations:
    alertmanager-type: Route
    alertmanager-default-route: "true"
data:
  spec: |-
    receiver: team-X-mails
```


TODO
====
* sort various lists to ensure we are not needlessly regerating a new ConfigMap.

LICENSE
=======
See [LICENSE](./LICENSE)
