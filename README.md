# oblik

Oblik is a Kubernetes operator designed to apply Vertical Pod Autoscaler (VPA) resource recommendations to workloads such as Deployments, StatefulSets, DaemonSets, CronJobs, and CNPG Clusters. Oblik runs on a cron-like schedule and can be configured via annotations on the workloads. It also provides a CLI for manual operations and includes a mutating webhook to enforce default resources.

**Oblik makes VPA compatible with HPA**; you can use the Horizontal Pod Autoscaler (HPA) as before. Oblik only handles resource definitions automatically using VPA recommendations.

**Summary**
- [How it works](#how-it-works)
- [Usage](#usage)
  - [Minimal Tuning Example](#minimal-tuning-example)
  - [Example: Uncapping Minimum Memory Limit](#example-uncapping-minimum-memory-limit)
  - [Applying the Workload](#applying-the-workload)
- [Features](#features)
- [Installation](#installation)
  - [Deploying Oblik with ArgoCD](#deploying-oblik-with-argocd)
- [Requirements](#requirements)
- [Configuration](#configuration)
  - [Annotations](#annotations)
  - [Targeting Specific Containers](#targeting-specific-containers)
  - [Recommendations:](#recommendations)
    - [Example](#example)
- [Using the CLI](#using-the-cli)
  - [CLI Usage](#cli-usage)
  - [Downloading the CLI](#downloading-the-cli)
  - [Docker Image](#docker-image)
- [Environment Variables](#environment-variables)
- [Limitations and overcoming them](#limitations-and-overcoming-them)
  - [VPA Recommendations for Certain Workloads](#vpa-recommendations-for-certain-workloads)
    - [Recommended Configurations](#recommended-configurations)
      - [1. Manually Set Memory Requests](#1-manually-set-memory-requests)
      - [2. Calculate Memory Based on CPU Usage Recommendations](#2-calculate-memory-based-on-cpu-usage-recommendations)
    - [Example Configuration for a JVM Application](#example-configuration-for-a-jvm-application)
  - [VPA Metrics Collection Period](#vpa-metrics-collection-period)
  - [High Resource Usage at Pod Startup](#high-resource-usage-at-pod-startup)
    - [Solutions](#solutions)
    - [Example: Uncapping Minimum Memory Limit](#example-uncapping-minimum-memory-limit-1)
    - [Adjusting Probes](#adjusting-probes)
- [Running Tests](#running-tests)
  - [Running Specific Tests](#running-specific-tests)
- [Contributing](#contributing)
- [Related Projects and Resources](#related-projects-and-resources)
  - [Additional Resources](#additional-resources)
- [License](#license)

## How it works

* **Automatic VPA Management**: Oblik automatically creates, updates, and deletes VPA objects for enabled workloads.
* **Cron-like Scheduling**: Oblik runs on a configurable cron schedule to apply VPA recommendations to workloads. You can specify the schedule using annotations on the workloads, and include random delays to stagger updates across your cluster.
* **Mutating Admission Webhook**: Oblik includes a mutating admission webhook that enforces resource requests and limits default policies on initial deployment of workloads and use eventually available recommendations from VPA on deployment updates.

## Usage

### Minimal Tuning Example

Here is a minimal tuning example using commonly used options such as `min-request-cpu` and `min-request-memory`:

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


### Applying the Workload

Apply your workload manifest as usual:

```sh
kubectl apply -f your-deployment.yaml
```

Oblik will automatically create a corresponding VPA object and manage resource recommendations according to the configured cron schedule and annotations.

## Features

* **Automatic VPA Management**: Oblik automatically creates, updates, and deletes VPA objects for enabled workloads.
* **Applies VPA Recommendations**: Automatically applies resource recommendations to workloads.
* **Configurable via Annotations**: Customize behavior using annotations on workloads.
* **Supports CPU and Memory Recommendations**: Adjust CPU and memory requests and limits.
* **Cron Scheduling with Random Delays**: Schedule updates with optional random delays to stagger them, avoiding a pods restart dance.
* **Supported Workload Types**:
    * Deployments
    * StatefulSets
    * DaemonSets
    * CronJobs
    * `postgresql.cnpg.io/Cluster` (see [CNPG issue](https://github.com/cloudnative-pg/cloudnative-pg/issues/2574#issuecomment-2159044747))
* **Customizable Algorithms**: Use different algorithms and values for calculating resource adjustments.
* **Mutating Webhook**: Enforces default resources on initial deployment and use recommendations if VPA exists.
* **Mattermost Webhook Notifications**: Notify on resource updates (should also work with Slack but not actually tested).
* **CLI for Manual Operations**: Provides a command-line interface for manual control.
* **High Availability**: Minimizes the risk of the mutating webhook blocking deployments. Only the leader runs background cron resource updates to prevent conflicts.


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

## Requirements

* **Vertical Pod Autoscaler (VPA) Operator**: Oblik requires the official Kubernetes VPA operator, which is not installed by default on Kubernetes clusters. You can find it here: [Official Kubernetes VPA Operator](https://github.com/kubernetes/autoscaler/tree/master/vertical-pod-autoscaler)
    
    * **Note**: You only need the **VPA recommender** component. The admission-controller and updater components are not required and can be omitted. This reduces the complexity and scalability issues of the VPA operator.

## Configuration

To enable Oblik on a workload, you need to add a **label** to your workload object (e.g., Deployment, StatefulSet):

* **`oblik.socialgouv.io/enabled`**: `"true"` or `"false"`. Defaults to `"false"`. Set to `"true"` to enable Oblik on the workload.

The operator uses **annotations** on workload objects to configure its behavior. All annotations should be prefixed with `oblik.socialgouv.io/`.

### Annotations

| Annotation Key (without prefix) | Description | Options | Default |
| --- | --- | --- | --- |
| `cron` | Cron expression to schedule when the recommendations are applied. | Any valid cron expression | `"0 2 * * *"` |
| `cron-add-random-max` | Maximum random delay added to the cron schedule. | Duration (e.g., `"120m"`) | `"120m"` |
| `dry-run` | If set to `"true"`, Oblik will simulate the updates without applying them. | `"true"`, `"false"` | `"false"` |
| `webhook-enabled` | Enable Mattermost webhook notifications on resource updates. | `"true"`, `"false"` | `"false"` |
| `request-cpu-apply-mode` | CPU request recommendation mode. | `"enforce"`, `"off"` | `"enforce"` |
| `request-memory-apply-mode` | Memory request recommendation mode. | `"enforce"`, `"off"` | `"enforce"` |
| `limit-cpu-apply-mode` | CPU limit apply mode. | `"enforce"`, `"off"` | `"enforce"` |
| `limit-memory-apply-mode` | Memory limit apply mode. | `"enforce"`, `"off"` | `"enforce"` |
| `limit-cpu-calculator-algo` | CPU limit calculator algorithm. | `"ratio"`, `"margin"` | `"ratio"` |
| `limit-memory-calculator-algo` | Memory limit calculator algorithm. | `"ratio"`, `"margin"` | `"ratio"` |
| `limit-cpu-calculator-value` | Value used by the CPU limit calculator algorithm. | Any numeric value | `"1"` |
| `limit-memory-calculator-value` | Value used by the memory limit calculator algorithm. | Any numeric value | `"1"` |
| `unprovided-apply-default-request-cpu` | Default CPU request if not provided by the VPA. | `"off"`, `"minAllowed"`, `"maxAllowed"`, or an arbitrary value (e.g., `"100m"`) | `"off"` |
| `unprovided-apply-default-request-memory` | Default memory request if not provided by the VPA. | `"off"`, `"minAllowed"`, `"maxAllowed"`, or an arbitrary value (e.g., `"128Mi"`) | `"off"` |
| `increase-request-cpu-algo` | Algorithm to increase CPU request. | `"ratio"`, `"margin"` | `"ratio"` |
| `increase-request-cpu-value` | Value used to increase CPU request. | Any numeric value | `"1"` |
| `increase-request-memory-algo` | Algorithm to increase memory request. | `"ratio"`, `"margin"` | `"ratio"` |
| `increase-request-memory-value` | Value used to increase memory request. | Any numeric value | `"1"` |
| `min-limit-cpu` | Minimum CPU limit value. | Any valid CPU value (e.g., `"200m"`) | "" |
| `max-limit-cpu` | Maximum CPU limit value. | Any valid CPU value (e.g., `"4"`) | "" |
| `min-limit-memory` | Minimum memory limit value. | Any valid memory value (e.g., `"200Mi"`) | "" |
| `max-limit-memory` | Maximum memory limit value. | Any valid memory value (e.g., `"8Gi"`) | "" |
| `min-request-cpu` | Minimum CPU request value. | Any valid CPU value (e.g., `"80m"`) | "" |
| `max-request-cpu` | Maximum CPU request value. | Any valid CPU value (e.g., `"8"`) | "" |
| `min-request-memory` | Minimum memory request value. | Any valid memory value (e.g., `"200Mi"`) | "" |
| `max-request-memory` | Maximum memory request value. | Any valid memory value (e.g., `"20Gi"`) | "" |
| `min-allowed-recommendation-cpu` | Minimum allowed CPU recommendation value. Overrides VPA `minAllowed.cpu`. | Any valid CPU value | "" |
| `max-allowed-recommendation-cpu` | Maximum allowed CPU recommendation value. Overrides VPA `maxAllowed.cpu`. | Any valid CPU value | "" |
| `min-allowed-recommendation-memory` | Minimum allowed memory recommendation value. Overrides VPA `minAllowed.memory`. | Any valid memory value | "" |
| `max-allowed-recommendation-memory` | Maximum allowed memory recommendation value. Overrides VPA `maxAllowed.memory`. | Any valid memory value | "" |
| `min-diff-cpu-request-algo` | Algorithm to calculate the minimum CPU request difference for applying recommendations. | `"ratio"`, `"margin"` | `"ratio"` |
| `min-diff-cpu-request-value` | Value used for minimum CPU request difference calculation. | Any numeric value | `"0"` |
| `min-diff-memory-request-algo` | Algorithm to calculate the minimum memory request difference for applying recommendations. | `"ratio"`, `"margin"` | `"ratio"` |
| `min-diff-memory-request-value` | Value used for minimum memory request difference calculation. | Any numeric value | `"0"` |
| `min-diff-cpu-limit-algo` | Algorithm to calculate the minimum CPU limit difference for applying recommendations. | `"ratio"`, `"margin"` | `"ratio"` |
| `min-diff-cpu-limit-value` | Value used for minimum CPU limit difference calculation. | Any numeric value | `"0"` |
| `min-diff-memory-limit-algo` | Algorithm to calculate the minimum memory limit difference for applying recommendations. | `"ratio"`, `"margin"` | `"ratio"` |
| `min-diff-memory-limit-value` | Value used for minimum memory limit difference calculation. | Any numeric value | `"0"` |
| `memory-request-from-cpu-enabled` | Calculate memory request from CPU request instead of recommendation. | `"true"`, `"false"` | `"false"` |
| `memory-limit-from-cpu-enabled` | Calculate memory limit from CPU limit instead of recommendation. | `"true"`, `"false"` | `"false"` |
| `memory-request-from-cpu-algo` | Algorithm to calculate memory request based on CPU request. | `"ratio"`, `"margin"` | `"ratio"` |
| `memory-request-from-cpu-value` | Value used for calculating memory request from CPU request. | Any numeric value | `"2"` |
| `memory-limit-from-cpu-algo` | Algorithm to calculate memory limit based on CPU limit. | `"ratio"`, `"margin"` | `"ratio"` |
| `memory-limit-from-cpu-value` | Value used for calculating memory limit from CPU limit. | Any numeric value | `"2"` |
| `request-apply-target` | Select which recommendation to apply by default on request. | `"frugal"`, `"balanced"`, `"peak"` | `"balanced"` |
| `request-cpu-apply-target` | Select which recommendation to apply for CPU request. | `"frugal"`, `"balanced"`, `"peak"` | `"balanced"` |
| `request-memory-apply-target` | Select which recommendation to apply for memory request. | `"frugal"`, `"balanced"`, `"peak"` | `"balanced"` |
| `limit-apply-target` | Select which recommendation to apply by default on limit. | `"auto"`, `"frugal"`, `"balanced"`, `"peak"` | `"auto"` |
| `limit-cpu-apply-target` | Select which recommendation to apply for CPU limit. | `"auto"`, `"frugal"`, `"balanced"`, `"peak"` | `"auto"` |
| `limit-memory-apply-target` | Select which recommendation to apply for memory limit. | `"auto"`, `"frugal"`, `"balanced"`, `"peak"` | `"auto"` |
| `request-cpu-scale-direction` | Allowed scaling direction for CPU request. | `"both"`, `"up"`, `"down"` | `"both"` |
| `request-memory-scale-direction` | Allowed scaling direction for memory request. | `"both"`, `"up"`, `"down"` | `"both"` |
| `limit-cpu-scale-direction` | Allowed scaling direction for CPU limit. | `"both"`, `"up"`, `"down"` | `"both"` |
| `limit-memory-scale-direction` | Allowed scaling direction for memory limit. | `"both"`, `"up"`, `"down"` | `"both"` |


### Targeting Specific Containers

To apply configurations to a specific container within a workload, suffix the annotation key with the container name, e.g.:

* **`oblik.socialgouv.io/min-limit-memory.hasura`**: Sets the minimum memory limit for the container named `hasura`.

### Recommendations:

* **Do not specify resource requests and limits in your workload manifest.** Let Oblik handle them based on VPA recommendations and settings as oblik annotation and default settings on operator deployment.
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

## Using the CLI

Oblik provides a CLI for manual operations. You can download the binary from the [GitHub releases](https://github.com/SocialGouv/oblik/releases).

### CLI Usage

* **Process all workloads in all namespaces**:
    
    ```sh
    oblik --all
    ```
    
* **Process a specific workload**:
    
    ```sh
    oblik --namespace my-ns --name example-deployment
    ```

* **Process workloads using selectors**:
    
    ```sh
    oblik --namespace my-ns --selector foo=bar
    ```


* **Force Mode**:
    
    Use the `--force` flag to run on workloads that do not have the `oblik.socialgouv.io/enabled: "true"` label.
    
    ```sh
    oblik --namespace my-ns --name example-deployment --force
    ```
    

### Downloading the CLI

You can download the latest CLI binary from the [GitHub releases](https://github.com/SocialGouv/oblik/releases).

### Docker Image

The Docker image for the Oblik operator is available and can be used to run the operator in your Kubernetes cluster.

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

## Limitations and overcoming them

### VPA Recommendations for Certain Workloads

The **Vertical Pod Autoscaler (VPA)** may not provide accurate memory recommendations for specific workloads, such as **Java Virtual Machines (JVM)** and **PostgreSQL** databases. These applications manage their own memory usage internally, which can lead to discrepancies in VPA's memory suggestions.

#### Recommended Configurations

For workloads where VPA's memory recommendations are unreliable, consider the following two options:

##### 1. Manually Set Memory Requests

Disable VPA’s automatic memory management and manually configure memory resources to suit your application’s needs.

**Steps:**

1. **Disable VPA Memory Handling:**
    * Set the `oblik.socialgouv.io/request-memory-apply-mode` annotation to `"off"` in your deployment.
2. **Manually Configure Memory Resources:**
    * Define memory requests and limits directly within your deployment specification.

**Example:**

```yaml
metadata:
  annotations:
    oblik.socialgouv.io/request-memory-apply-mode: "off"
spec:
  containers:
    - name: jvm-container
      image: your-jvm-image:latest
      resources:
        requests:
          memory: "3Gi"  # Manually set based on application requirements
      ports:
        - containerPort: 8080
```

##### 2. Calculate Memory Based on CPU Usage Recommendations

Leverage the relationship between CPU and memory usage to derive memory allocations from CPU recommendations. This method is particularly effective for applications like PostgreSQL, where a typical memory-to-CPU ratio is well-established.

**Example Ratio:**

* **PostgreSQL:** For every 1 CPU, allocate approximately 4 GB of memory.

**Steps:**

1. **Enable Memory Calculation from CPU:**
    * Set `oblik.socialgouv.io/memory-request-from-cpu-enabled` to `"true"`.

2. **Specify the Calculation Algorithm and Ratio:**
    * Use `memory-request-from-cpu-algo` set to `"ratio"`.
    * Define the ratio with `memory-request-from-cpu-value`. For instance, a value of `"4"` implies 4 GB of memory per CPU.

**Example:**

```yaml
metadata:
  annotations:
    oblik.socialgouv.io/memory-request-from-cpu-enabled: "true"
    oblik.socialgouv.io/memory-request-from-cpu-algo: "ratio"
    oblik.socialgouv.io/memory-request-from-cpu-value: "4"
```

#### Example Configuration for a JVM Application

Below is an example YAML configuration for deploying a JVM application using CPU-based memory calculation:

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
    oblik.socialgouv.io/memory-request-from-cpu-value: "2"
spec:
  replicas: 3
  selector:
    matchLabels:
      app: jvm-app
  template:
    metadata:
      labels:
        app: jvm-app
    spec:
      containers:
        - name: jvm-container
          image: your-jvm-image:latest
          resources:
            requests:
              cpu: "1"
              memory: "2Gi"  # Derived from CPU usage (1 CPU * 2 GB)
            limits:
              cpu: "2"
              memory: "4Gi"
          ports:
            - containerPort: 8080
```

**Explanation:**

* **Annotations:**
    
    * `memory-request-from-cpu-enabled: "true"`: Enables memory calculation based on CPU usage.
    * `memory-request-from-cpu-algo: "ratio"`: Specifies that a fixed ratio will be used for calculation.
    * `memory-request-from-cpu-value: "2"`: Sets the memory allocation to 2 GB per CPU.
* **Resources:**
    
    * **Requests:**
        * `cpu: "1"`: Requests 1 CPU.
        * `memory: "2Gi"`: Allocates 2 GB of memory based on the ratio (1 CPU * 2 GB).
    * **Limits:**
        * `cpu: "2"`: Limits the container to 2 CPUs.
        * `memory: "4Gi"`: Limits the memory usage to 4 GB.


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
* [Stop Using CPU Limits](https://home.robusta.dev/blog/stop-using-cpu-limits)
* [Why You Should Keep Using CPU Limits on Kubernetes](https://dnastacio.medium.com/why-you-should-keep-using-cpu-limits-on-kubernetes-60c4e50dfc61)

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
