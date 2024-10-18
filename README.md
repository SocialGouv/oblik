# oblik

Oblik is a Kubernetes operator designed to apply Vertical Pod Autoscaler (VPA) resource recommendations to workloads such as Deployments, StatefulSets, DaemonSets, CronJobs, and CNPG Clusters. Oblik runs on a cron-like schedule and can be configured via annotations on the workloads. It also provides a CLI for manual operations and includes a mutating webhook to enforce default resources.

**Oblik makes VPA compatible with HPA**; you can use the Horizontal Pod Autoscaler (HPA) as before. Oblik only handles resource definitions automatically using VPA recommendations.

## How it works

* **Automatic VPA Management**: Oblik automatically creates, updates, and deletes VPA objects for enabled workloads.
* **Cron-like Scheduling**: Oblik runs on a configurable cron schedule to apply VPA recommendations to workloads. You can specify the schedule using annotations on the workloads, and include random delays to stagger updates across your cluster.
* **Mutating Admission Webhook**: Oblik includes a mutating admission webhook that enforces default resource requests and limits on initial deployment of workloads. The webhook applies default resource requests and limits if they are not specified in the workload's manifest.

## Requirements

* **Vertical Pod Autoscaler (VPA) Operator**: Oblik requires the official Kubernetes VPA operator, which is not installed by default on Kubernetes clusters. You can find it here: [Official Kubernetes VPA Operator](https://github.com/kubernetes/autoscaler/tree/master/vertical-pod-autoscaler)
    
    * **Note**: You only need the **VPA recommender** component. The admission-controller and updater components are not required and can be omitted. This reduces the complexity and scalability issues of the VPA operator.
            

## Installation

You can deploy Oblik using the provided Helm chart:

```shell
git clone https://github.com/SocialGouv/oblik.git
cd oblik/charts/oblik
helm upgrade --install oblik . --namespace oblik
```

Alternatively, you can use the Docker image for the operator, and download the CLI binary from the [GitHub releases](https://github.com/SocialGouv/oblik/releases).

### Deploying Oblik with ArgoCD

If you're using ArgoCD to manage your Kubernetes deployments, to integrate the Oblik Helm chart you need to include specific settings in your ArgoCD `Application` resource.

Add the following sections to your ArgoCD `Application` spec:

```yaml
spec:
  syncPolicy:
    syncOptions:
      - ApplyOutOfSyncOnly=true
      - RespectIgnoreDifferences=true
  ignoreDifferences:
    - group: admissionregistration.k8s.io
      kind: MutatingWebhookConfiguration
      name: oblik
      jsonPointers:
        - /webhooks/0/clientConfig/caBundle
    - group: ""
      kind: Secret
      name: webhook-certs
      jsonPointers:
        - /data
```

## Features

* **Automatic VPA Management**: Oblik automatically creates, updates, and deletes VPA objects for enabled workloads.
* **Applies VPA Recommendations**: Automatically applies resource recommendations to workloads.
* **Configurable via Annotations**: Customize behavior using annotations on workloads.
* **Supports CPU and Memory Recommendations**: Adjust CPU and memory requests and limits.
* **Cron Scheduling with Random Delays**: Schedule updates with optional random delays to stagger them.
* **Supported Workload Types**:
    * Deployments
    * StatefulSets
    * DaemonSets
    * CronJobs
    * `postgresql.cnpg.io/Cluster` (see [CNPG issue](https://github.com/cloudnative-pg/cloudnative-pg/issues/2574#issuecomment-2155389267))
* **Customizable Algorithms**: Use different algorithms and values for calculating resource adjustments.
* **Mutating Webhook**: Enforces default resources on initial deployment and applies recommendations if VPA exists.
* **Mattermost Webhook Notifications**: Notify on resource updates (should also work with Slack but not actually tested).
* **CLI for Manual Operations**: Provides a command-line interface for manual control.

## Limitations

### VPA Recommendations for Certain Workloads

The VPA may not provide relevant memory recommendations for some workloads, such as Java Virtual Machines (JVM) and PostgreSQL databases. These applications manage their own memory usage internally, which can lead to inaccurate recommendations from the VPA.

#### Recommended Configuration

For such workloads, it's advisable to calculate memory limits from requests to maintain stability, rather than disabling memory limit adjustments.

* **Calculate Memory Limit from Request**:
    
    ```yaml
    oblik.socialgouv.io/limit-memory-calculator-algo: "ratio"
    oblik.socialgouv.io/limit-memory-calculator-value: "1"
    ```
    
* **Calculate CPU Limit from Request**:
    
    ```yaml
    oblik.socialgouv.io/limit-cpu-calculator-algo: "ratio"
    oblik.socialgouv.io/limit-cpu-calculator-value: "1.5"
    ```
    

#### Example for a JVM Application

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: jvm-app
  namespace: default
  labels:
    oblik.socialgouv.io/enabled: "true"
  annotations:
    oblik.socialgouv.io/memory-request-from-cpu-enabled: "true"
    oblik.socialgouv.io/memory-request-from-cpu-algo: "ratio"
    oblik.socialgouv.io/memory-request-from-cpu-value: "4"
    oblik.socialgouv.io/limit-memory-calculator-algo: "ratio"
    oblik.socialgouv.io/limit-memory-calculator-value: "1"
    oblik.socialgouv.io/limit-cpu-calculator-algo: "ratio"
    oblik.socialgouv.io/limit-cpu-calculator-value: "1.5"
spec:
  # ... your deployment spec
```

### VPA Metrics Collection Period

The VPA bases its recommendations on the last **8 days** of Prometheus metrics. If your workload has irregular loads with intervals longer than this 8-day period, the VPA (and thus Oblik) may not provide relevant recommendations for these workloads.

For workloads with irregular patterns beyond the 8-day window, consider manually managing resource requests and limits or using alternative scaling strategies.

### High Resource Usage at Pod Startup

Some applications may have high CPU or memory usage during pod startup, which can cause issues with readiness and startup probes.

#### Solutions

* **Adjust Startup Probes**: Increase the initial delay or timeout for startup and readiness probes to accommodate the higher resource usage at startup.
    
* **Increase Resource Limits or Recommendations**: Increase the CPU and memory limits or recommendations to provide more resources during startup.
    

#### Example: Uncapping Minimum Memory Limit

If your application requires higher memory consumption during pod startup, you might need to uncap the `min-limit-memory` to a value higher than the request.

* **Using Hardcoded Value**:
    
    ```yaml
    oblik.socialgouv.io/min-limit-memory: "512Mi"
    ```
    
* **Using Calculator Algorithm**:
    
    ```yaml
    oblik.socialgouv.io/limit-memory-calculator-algo: "margin"
    oblik.socialgouv.io/limit-memory-calculator-value: "256Mi"
    ```
    

#### Adjusting Probes

```yaml
spec:
  containers:
    - name: app-container
      image: your-image
      readinessProbe:
        initialDelaySeconds: 30
        timeoutSeconds: 5
      startupProbe:
        initialDelaySeconds: 60
        timeoutSeconds: 10
```

## Configuration

To enable Oblik on a workload, you need to add a **label** to your workload object (e.g., Deployment, StatefulSet):

* **`oblik.socialgouv.io/enabled`**: `"true"` or `"false"`. Defaults to `"false"`. Set to `"true"` to enable Oblik on the workload.

The operator uses **annotations** on workload objects to configure its behavior. All annotations should be prefixed with `oblik.socialgouv.io/`.

### Annotations

[Please refer to the annotations table in the documentation for detailed configuration options.]

### Targeting Specific Containers

To apply configurations to a specific container within a workload, suffix the annotation key with the container name, e.g.:

* **`oblik.socialgouv.io/min-limit-memory.hasura`**: Sets the minimum memory limit for the container named `hasura`.

### Mutating Admission Webhook

Oblik includes a mutating admission webhook that enforces default resources on the initial deployment of a workload. The webhook applies default resource requests and limits if they are not specified in the workload's manifest.

**Recommendations:**

* **Do not specify resource requests and limits in your workload manifest.** Let Oblik handle them based on VPA recommendations and default settings.
* The webhook will read the VPA if it exists and apply recommendations to the workload upon deployment.

#### Example

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: example-deployment
  namespace: default
  labels:
    oblik.socialgouv.io/enabled: "true"
  annotations:
    oblik.socialgouv.io/min-request-cpu: "100m"
    oblik.socialgouv.io/min-request-memory: "128Mi"
spec:
  # Do not specify resources; let Oblik handle them
  containers:
    - name: app-container
      image: your-image
```

## Usage

### Minimal Example

Here is a minimal example using commonly used options such as `min-request-cpu` and `min-request-memory`:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: minimal-app
  namespace: default
  labels:
    oblik.socialgouv.io/enabled: "true"
  annotations:
    oblik.socialgouv.io/min-request-cpu: "100m"
    oblik.socialgouv.io/min-request-memory: "128Mi"
spec:
  # Do not specify resources; let Oblik handle them
  containers:
    - name: app-container
      image: your-image
```

### Example: Uncapping Minimum Memory Limit

If your application requires higher memory consumption during pod startup, you might need to uncap the `min-limit-memory` to a value higher than the request.

* **Using Hardcoded Value**:
    
    ```yaml
    oblik.socialgouv.io/min-limit-memory: "512Mi"
    ```
    
* **Using Calculator Algorithm**:
    
    ```yaml
    oblik.socialgouv.io/limit-memory-calculator-algo: "margin"
    oblik.socialgouv.io/limit-memory-calculator-value: "256Mi"
    ```
    

### Adjusting Startup Probes

For applications with high CPU usage at startup, you may need to adjust the startup and readiness probes:

```yaml
spec:
  containers:
    - name: app-container
      image: your-image
      readinessProbe:
        initialDelaySeconds: 30
        timeoutSeconds: 5
      startupProbe:
        initialDelaySeconds: 60
        timeoutSeconds: 10
```

Alternatively, you can increase the CPU limit or CPU recommendation to provide more resources during startup.

### Applying the Workload

Apply your workload manifest as usual:

```sh
kubectl apply -f your-deployment.yaml
```

Oblik will automatically create a corresponding VPA object and manage resource recommendations according to the configured cron schedule and annotations.

## Using the CLI

Oblik provides a CLI for manual operations. You can download the binary from the [GitHub releases](https://github.com/SocialGouv/oblik/releases).

### CLI Usage

```plaintext
Oblik CLI

Usage:
  oblik [flags]
  oblik [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  help        Help about any command
  operator    Oblik operator

Flags:
  -a, --all                Process all namespaces
  -f, --force              Force to run on not enabled workloads
  -h, --help               help for oblik
      --name string        Name of the workload
  -n, --namespace string   Namespace containing workloads
  -l, --selector string    Label selector for filtering workloads
  -v, --version            Show version
```

### Examples

* **Process all workloads in all namespaces**:
    
    ```sh
    oblik --all
    ```
    
* **Process a specific workload**:
    
    ```sh
    oblik --namespace default --name example-deployment
    ```
    
* **Dry-run mode**:
    
    Add the `dry-run` annotation to simulate updates without applying them.
    
* **Force Mode**:
    
    Use the `--force` flag to run on workloads that do not have the `oblik.socialgouv.io/enabled: "true"` label.
    
    ```sh
    oblik --namespace default --name example-deployment --force
    ```
    

### Downloading the CLI

You can download the latest CLI binary from the [GitHub releases](https://github.com/SocialGouv/oblik/releases).

### Docker Image

The Docker image for the Oblik operator is available and can be used to run the operator in your Kubernetes cluster.

### Helm Charts

Helm charts are provided in the `charts/oblik` directory of the repository for easy deployment.

## Environment Variables

The Oblik Kubernetes VPA Operator uses the following environment variables for configuration. These environment variables allow you to set default values and customize the behavior of the operator.

| Environment Variable | Description | Options | Default |
| --- | --- | --- | --- |
| `OBLIK_DEFAULT_CRON` | Default cron expression for scheduling when the recommendations are applied. | Any valid cron expression | `"0 2 * * *"` |
| `OBLIK_DEFAULT_CRON_ADD_RANDOM_MAX` | Maximum random delay added to the cron schedule. | Duration (e.g., `"120m"`) | `"120m"` |
| `OBLIK_DEFAULT_LIMIT_CPU_CALCULATOR_ALGO` | Default algorithm to use for calculating CPU limits. | `"ratio"`, `"margin"` | `"ratio"` |
| `OBLIK_DEFAULT_LIMIT_MEMORY_CALCULATOR_ALGO` | Default algorithm to use for calculating memory limits. | `"ratio"`, `"margin"` | `"ratio"` |
| `OBLIK_DEFAULT_LIMIT_CPU_CALCULATOR_VALUE` | Default value to use with the CPU limit calculator algorithm. | Any numeric value | `"1"` |
| `OBLIK_DEFAULT_LIMIT_MEMORY_CALCULATOR_VALUE` | Default value to use with the memory limit calculator algorithm. | Any numeric value | `"1"` |
| `OBLIK_DEFAULT_UNPROVIDED_APPLY_DEFAULT_REQUEST_CPU` | Default behavior for CPU requests if not provided. | `"off"`, `"minAllow"`, `"maxAllow"`, or value | `"off"` |
| `OBLIK_DEFAULT_UNPROVIDED_APPLY_DEFAULT_REQUEST_MEMORY` | Default behavior for memory requests if not provided. | `"off"`, `"minAllow"`, `"maxAllow"`, or value | `"off"` |
| `OBLIK_DEFAULT_INCREASE_REQUEST_CPU_ALGO` | Default algorithm to use for increasing CPU requests. | `"ratio"`, `"margin"` | `"ratio"` |
| `OBLIK_DEFAULT_INCREASE_REQUEST_MEMORY_ALGO` | Default algorithm to use for increasing memory requests. | `"ratio"`, `"margin"` | `"ratio"` |
| `OBLIK_DEFAULT_INCREASE_REQUEST_CPU_VALUE` | Default value to use with the algorithm for increasing CPU requests. | Any numeric value | `"1"` |
| `OBLIK_DEFAULT_INCREASE_REQUEST_MEMORY_VALUE` | Default value to use with the algorithm for increasing memory requests. | Any numeric value | `"1"` |
| `OBLIK_DEFAULT_MIN_LIMIT_CPU` | Value used to cap minimum CPU limit. | Any valid CPU value | `""` |
| `OBLIK_DEFAULT_MAX_LIMIT_CPU` | Value used to cap maximum CPU limit. | Any valid CPU value | `""` |
| `OBLIK_DEFAULT_MIN_LIMIT_MEMORY` | Value used to cap minimum memory limit. | Any valid memory value | `""` |
| `OBLIK_DEFAULT_MAX_LIMIT_MEMORY` | Value used to cap maximum memory limit. | Any valid memory value | `""` |
| `OBLIK_DEFAULT_MIN_REQUEST_CPU` | Value used to cap minimum CPU request. | Any valid CPU value | `""` |
| `OBLIK_DEFAULT_MAX_REQUEST_CPU` | Value used to cap maximum CPU request. | Any valid CPU value | `""` |
| `OBLIK_DEFAULT_MIN_REQUEST_MEMORY` | Value used to cap minimum memory request. | Any valid memory value | `""` |
| `OBLIK_DEFAULT_MAX_REQUEST_MEMORY` | Value used to cap maximum memory request. | Any valid memory value | `""` |
| `OBLIK_MATTERMOST_WEBHOOK_URL` | Webhook URL for Mattermost notifications. | URL | `""` |

## Links

* ["Stop Using CPU Limits"](https://home.robusta.dev/blog/stop-using-cpu-limits)
* ["Why You Should Keep Using CPU Limits on Kubernetes"](https://dnastacio.medium.com/why-you-should-keep-using-cpu-limits-on-kubernetes-60c4e50dfc61)

## Running Tests

The Oblik project includes end-to-end tests that can be run to verify the functionality of the operator. These tests are located in the `tests` directory and are implemented using Go's testing framework.

To run all tests, use the following command from the root of the project:

```sh
go test ./tests -v
```

### Running Specific Tests

You can run a specific test case by using the `-test-case` flag. This flag allows you to specify the name of a single test case to run, which is particularly useful when debugging or focusing on a particular feature.

To run a specific test, use the following command:

```sh
go test ./tests -v -test-case=TestCaseName
```

Replace `TestCaseName` with the name of the test case you want to run. For example, to run the "TestOffRecommendations" test case:

```sh
go test ./tests -v -test-case=TestOffRecommendations
```

This will run only the specified test case, allowing for faster and more focused testing during development or debugging.

## Contributing

We welcome contributions! Please feel free to submit pull requests or open issues on our GitHub repository.

## Related Projects and Resources

* [Predictive Horizontal Pod Autoscaler](https://github.com/jthomperoo/predictive-horizontal-pod-autoscaler)
* [Custom Pod Autoscaler](https://github.com/jthomperoo/custom-pod-autoscaler)
* [kube-reqsizer](https://github.com/ElementTech/kube-reqsizer/)
    * [Reddit Discussion](https://www.reddit.com/r/kubernetes/comments/11vhfbz/kubereqsizer_open_source_vpa_alternative_for/)
* [HVPA Controller](https://github.com/gardener/hvpa-controller)
    * The initial approach of Oblik was to find a balance between HPA and VPA, but we changed our strategy.

### Additional Resources

* [Vertical Pod Autoscaler Deep Dive: Limitations and Real-world Examples](https://medium.com/infrastructure-adventures/vertical-pod-autoscaler-deep-dive-limitations-and-real-world-examples-9195f8422724)
* [Vertical Pod Autoscaler (VPA): Know Everything About It](https://foxutech.com/vertical-pod-autoscalervpa-know-everything-about-it/)
* [Performance Evaluation of the Autoscaling Strategies (Vertical and Horizontal) Using Kubernetes](https://medium.com/@kewynakshlley/performance-evaluation-of-the-autoscaling-strategies-vertical-and-horizontal-using-kubernetes-42d9a1663e6b)
* [Kubernetes HPA Custom Metrics for Effective CPU/Memory Scaling](https://caiolombello.medium.com/kubernetes-hpa-custom-metrics-for-effective-cpu-memory-scaling-23526bba9b4)
* [Kubernetes Autoscaling Concepts](https://kubernetes.io/docs/concepts/workloads/autoscaling/)
* [11 Ways to Optimize Kubernetes Vertical Pod Autoscaler](https://overcast.blog/11-ways-to-optimize-kubernetes-vertical-pod-autoscaler-930246954fc4)
* [Multidimensional Pod Autoscaler - AEP](https://github.com/kubernetes/autoscaler/blob/master/multidimensional-pod-autoscaler/AEP.md)
* [Google Cloud: Multidimensional Pod Autoscaling](https://cloud.google.com/kubernetes-engine/docs/how-to/multidimensional-pod-autoscaling)

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
