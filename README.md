# argo-events-cronjob-trigger

An [Argo Events Trigger](https://argoproj.github.io/argo-events/concepts/trigger/) to trigger a Kubernetes `CronJob`.

## Background

It is quite easy to manually trigger a Kubernetes `CronJob`, e.g. via `kubectl create job hello-job --from=cronjob/hello-cronjob`.
But doing this via Argo Events proved cumbersome, as the built-in K8s Resources trigger seems to lack support for `--from`.

Hence, we built this little trigger which creates a `Job` based on the `spec.jobTemplate` of any existing K8s `CronJob`.

## Installation

### Base

We recommend installing using [Kustomize](https://kustomize.io/):

```shell
cd kustomize/base
kustomize edit set image cronjob-trigger=ghcr.io/j-be/argo-events-cronjob-trigger:<tag>
kubectl apply -k . -n <namespace>
```

where

* `<tag>` is the latest available version [here](https://github.com/j-be/argo-events-cronjob-trigger/pkgs/container/argo-events-cronjob-trigger)
* `<namespace>` is the Kubernetes Namespace you want to deploy the trigger to.
  This may be the same namespace that Argo Events is deployed to, but it doesn't have to be.

The trigger provides a `Service` running on `http://cronjob-trigger:9000`.
 See [here](./kustomize/base/service.yaml) for details.

### RBAC

The base installation creates a `ServiceAccount` as well as a `ClusterRole` which allows to `get` `cronjobs` and `create` `jobs`.
See [here](./kustomize/base/serviceaccount.yaml) for details.

But there is neither a `ClusterRoleBinding` nor `RoleBinding` included by default.
In the spirit of "least privilege" we recommend creating `RoleBinding`s in the namespaces the `CronJob`s to be triggerd reside.
An example would be:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: cronjob-trigger-role-binding
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: cronjob-trigger-role
subjects:
  - kind: ServiceAccount
    name: cronjob-trigger-sa
    namespace: <namespace>
```

where`<namespace>` is the Kubernetes Namespace the trigger was deployed to.
See [Base](#base) for details.

## Usage

We assume you have Argo Events setup and working.
If not, please consult the [Argo Events Operator Manual](https://argoproj.github.io/argo-events/installation/).

We also assume you have an `EventSource` set up in Argo Events
If not, please consult the [Argo Events User Guide](https://argoproj.github.io/argo-events/concepts/architecture/).

To trigger a `CronJob` named `hello-cronjob` located in the namespace `hello-namespace` create the following `Sensor`:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Sensor
metadata:
  name: hello-cronjob-sensor
spec:
  dependencies:
    - name: hello
      eventSourceName: hello-eventsource
      eventName: hello-event
  triggers:
    - template:
        name: hello-trigger
        custom:
          serverURL: cronjob-trigger.<namespace>.svc:9000
          spec:
            namespace: hello-namespace
            cronjob: hello-cronjob
```

where `<namespace>` is the Kubernetes Namespace the trigger was deployed to.
See [Base](#base) for details.

For further details on `Sensor` please consult the [Argo Events User Guide](https://argoproj.github.io/argo-events/concepts/sensor/).
