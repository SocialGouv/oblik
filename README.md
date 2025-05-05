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
  - [Installing from the OCI Registry](#installing-from-the-oci-registry)
  - [Installing from Source](#installing-from-source)
  - [Prerequisites](#prerequisites)
    - [Installing VPA](#installing-vpa)
  - [Configuration Options](#configuration-options)
  - [Deploying Oblik with ArgoCD](#deploying-oblik-with-argocd)
- [Requirements](#requirements)
- [Configuration](#configuration)
  - [Logging Levels](#logging-levels)
  - [Configuration with Annotations](#configuration-with-annotations)
  - [Targeting Specific Containers](#targeting-specific-containers)
  - [Recommendations:](#recommendations)
    - [Example](#example)
- [ResourcesConfig CRD](#resourcesconfig-crd)
  - [Overview](#overview)
  - [When to Use ResourcesConfig vs. Annotations](#when-to-use-resourcesconfig-vs-annotations)
  - [Configuration Reference](#configuration-reference)
    - [1. Basic Configuration](#1-basic-configuration)
    - [2. CPU Request Settings](#2-cpu-request-settings)
    - [3. Memory Request Settings](#3-memory-request-settings)
    - [4. CPU Limit Settings](#4-cpu-limit-settings)
    - [5. Memory Limit Settings](#5-memory-limit-settings)
    - [6. Cross-Resource Calculation](#6-cross-resource-calculation)
    - [7. Minimum Difference Thresholds](#7-minimum-difference-thresholds)
    - [8. Container-Specific Configurations](#8-container-specific-configurations)
  - [Example Usage](#example-usage)
    - [Complete ResourcesConfig Example:](#complete-resourcesconfig-example)
    - [Comparison: Annotations vs. ResourcesConfig](#comparison-annotations-vs-resourcesconfig)
- [Using the CLI](#using-the-cli)
  - [CLI Usage](#cli-usage)
  - [Downloading the CLI](#downloading-the-cli)
  - [Docker Image](#docker-image)
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

Oblik can be installed using Helm in several ways:

### Installing from the OCI Registry

The recommended way to install Oblik is using the Helm chart from the OCI registry:

```shell
# Add the Oblik Helm repository
helm install oblik oci://ghcr.io/socialgouv/helm/oblik --version 0.1.0 --namespace oblik --create-namespace
```

### Installing from Source

Alternatively, you can clone the repository and install from the local chart:

```shell
git clone https://github.com/SocialGouv/oblik.git
cd oblik/charts/oblik
helm upgrade --install oblik . --namespace oblik --create-namespace
```

### Prerequisites

* **Kubernetes**: only tested from version 1.27 and highers
* **Vertical Pod Autoscaler (VPA)**: Oblik requires the VPA recommender component to function properly. See the [official VPA installation documentation](https://github.com/kubernetes/autoscaler/tree/master/vertical-pod-autoscaler#installation) for more details.

#### Installing VPA

You can install the VPA recommender using the included Helm chart. Before installing, you need to generate the required certificates:

```shell
# Clone the repository if you haven't already
git clone https://github.com/SocialGouv/oblik.git
cd oblik/charts/vpa

# Generate certificates for the VPA admission controller
./gencerts.sh

# Install the VPA components
helm install vpa . --namespace vpa --create-namespace
```

Alternatively, you can install from the OCI registry:

```shell
# First, download and run the gencerts.sh script
curl -O https://raw.githubusercontent.com/SocialGouv/oblik/main/charts/vpa/gencerts.sh
chmod +x gencerts.sh
./gencerts.sh

# Then install the VPA chart
helm install vpa oci://ghcr.io/socialgouv/helm/vpa --version 0.1.0 --namespace vpa --create-namespace
```

Note: The VPA chart included with Oblik is configured to install only the necessary components required for Oblik to function properly.

### Configuration Options

The Oblik Helm chart supports the following configuration options:

| Parameter | Description | Default |
|-----------|-------------|---------|
| `replicas` | Number of Oblik operator replicas | `3` |
| `image.repository` | Oblik image repository | `ghcr.io/socialgouv/oblik` |
| `image.tag` | Oblik image tag | Latest release |
| `image.pullPolicy` | Image pull policy | `IfNotPresent` |
| `webhook.enabled` | Enable mutating webhook | `true` |
| `webhook.failurePolicy` | Webhook failure policy | `Fail` |
| `args` | Additional arguments for the operator | `[]` |
| `env` | Environment variables for the operator | `{}` |
| `existingSecret` | Name of existing secret to use | `""` |
| `resources` | Resource requests and limits | `{}` |
| `annotations` | Annotations to add to the deployment | `{}` |

Example `values.yaml`:

```yaml
replicas: 2

image:
  repository: ghcr.io/socialgouv/oblik
  tag: latest
  pullPolicy: Always

webhook:
  enabled: true
  failurePolicy: Ignore

args:
  - "-v"
  - "2"

resources:
  requests:
    cpu: 100m
    memory: 128Mi
  limits:
    cpu: 500m
    memory: 256Mi
```

To install with custom values:

```shell
helm install oblik oci://ghcr.io/socialgouv/helm/oblik --version 0.1.0 --namespace oblik --create-namespace -f values.yaml
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

### Logging Levels

Oblik uses klog for logging and supports different verbosity levels that can be enabled when running the operator:

* **Level 0**: Info level logging (default)

* **Level 2**: Debug level logging

* **Level 3**: More verbose debug logging

To enable debug logging, set the appropriate verbosity level when running the operator:

```shell
# Enable debug logging
helm upgrade --install oblik . --namespace oblik --set args[0]="-v" --set args[1]="2"

# Enable verbose debug logging
helm upgrade --install oblik . --namespace oblik --set args[0]="-v" --set args[1]="3"
```

You can also set the verbosity level in your Helm values.yaml:

```yaml
args:
  - "-v=2"  # For debug logging
```

### Configuration with Annotations

Oblik can be configured using annotations on workloads. All available annotation options are documented in the [Configuration Reference](#configuration-reference) section under the ResourcesConfig CRD documentation. The annotation keys use kebab-case format (e.g., `oblik.socialgouv.io/min-request-cpu`) while the equivalent ResourcesConfig fields use camelCase (e.g., `minRequestCpu`).


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

## ResourcesConfig CRD

The ResourcesConfig CRD provides a Kubernetes-native way to configure Oblik's resource management behavior for specific workloads. Unlike annotations that are applied directly to workloads, ResourcesConfig is a separate resource that targets workloads using a reference.

### Overview

ResourcesConfig is a Custom Resource Definition (CRD) that allows you to define resource management policies for Kubernetes workloads in a more structured and maintainable way. It supports all the same configuration options as annotations, but uses camelCase field names instead of kebab-case annotation keys.

Key benefits of using ResourcesConfig:
- Separate resource management configuration from workload definitions
- Kubernetes-native approach with full YAML/JSON schema validation
- Ability to version control resource configurations independently
- Cleaner workload manifests without numerous annotations

### When to Use ResourcesConfig vs. Annotations

- **Use ResourcesConfig when**:
  - You want to manage resource configurations separately from workload definitions
  - You prefer a more Kubernetes-native approach with separate resources

- **Use Annotations when**:
  - You want to keep all configuration in the workload manifest
  - You're making simple, workload-specific adjustments
  - You prefer the simplicity of annotating existing resources

### Configuration Reference

The ResourcesConfig CRD fields use camelCase versions of the annotation keys. For example:

* **Annotation:** `oblik.socialgouv.io/min-request-cpu`
* **CRD Field:** `minRequestCpu`

Below are the configuration options organized by category:

* * *

#### 1. Basic Configuration

| Annotation Key | ResourcesConfig Field | Description | Options | Default |
| --- | --- | --- | --- | --- |
| N/A | `targetRef` | Points to the controller managing the set of pods. Must be an object with `kind`, `name`, and an optional `apiVersion`. | Object with kind, name, and optional apiVersion | **Required** |
| `cron` | `cron` | Cron expression to schedule when the recommendations are applied. Accepts any valid cron expression (e.g., `"0 2 * * *"`). | Any valid cron expression | `"0 2 * * *"` |
| `cron-add-random-max` | `cronAddRandomMax` | Maximum random delay added to the cron schedule. Accepts duration values (e.g., `"120m"`). | Duration (e.g., `"120m"`) | `"120m"` |
| `dry-run` | `dryRun` | If set to `"true"`, Oblik will simulate the updates without applying them. | `"true"`, `"false"` | `"false"` |
| `webhook-enabled` | `webhookEnabled` | Enable mutating webhook resources enforcement. | `"true"`, `"false"` | `"true"` |
| `annotation-mode` | `annotationMode` | Controls how annotations are managed. | `"replace"`, `"merge"` | `"replace"` |
| `unprovided-apply-default-request-cpu` | `unprovidedApplyDefaultRequestCpu` | Default CPU request if not provided by the VPA. **Overrides VPA** values (`minAllowed.cpu`/`maxAllowed.cpu`) when applicable. Accepts `"off"`, `"minAllowed"`, `"maxAllowed"`, or an arbitrary value (e.g., `"100m"`). | `"off"`, `"minAllowed"`, `"maxAllowed"`, or an arbitrary value (e.g., `"100m"`) | `"off"` |
| `unprovided-apply-default-request-memory` | `unprovidedApplyDefaultRequestMemory` | Default memory request if not provided by the VPA. **Overrides VPA** values (`minAllowed.memory`/`maxAllowed.memory`) when applicable. Accepts `"off"`, `"minAllowed"`, `"maxAllowed"`, or an arbitrary value (e.g., `"128Mi"`). | `"off"`, `"minAllowed"`, `"maxAllowed"`, or an arbitrary value (e.g., `"128Mi"`) | `"off"` |

* * *

#### 2. CPU Request Settings

| Annotation Key | ResourcesConfig Field | Description | Options | Default |
| --- | --- | --- | --- | --- |
| `request-apply-target` | `requestApplyTarget` | Select which recommendation to apply by default on request. | `"frugal"`, `"balanced"`, `"peak"` | `"balanced"` |
| `request-cpu-apply-mode` | `requestCpuApplyMode` | CPU request recommendation mode. | `"enforce"`, `"off"` | `"enforce"` |
| `min-request-cpu` | `minRequestCpu` | Minimum CPU request value. Accepts any valid CPU value (e.g., `"80m"`). | Any valid CPU value | `""` |
| `max-request-cpu` | `maxRequestCpu` | Maximum CPU request value. Accepts any valid CPU value (e.g., `"8"`) | Any valid CPU value | `""` |
| `request-cpu-apply-target` | `requestCpuApplyTarget` | Select which recommendation to apply for CPU request. | `"frugal"`, `"balanced"`, `"peak"` | `"balanced"` |
| `request-cpu-scale-direction` | `requestCpuScaleDirection` | Allowed scaling direction for CPU request. | `"both"`, `"up"`, `"down"` | `"both"` |
| `min-allowed-recommendation-cpu` | `minAllowedRecommendationCpu` | Minimum allowed CPU recommendation value. **Overrides VPA** `minAllowed.cpu`. Accepts any valid CPU value (e.g., `"80m"`). | Any valid CPU value | `""` |
| `max-allowed-recommendation-cpu` | `maxAllowedRecommendationCpu` | Maximum allowed CPU recommendation value. **Overrides VPA** `maxAllowed.cpu`. Accepts any valid CPU value (e.g., `"8"`). | Any valid CPU value | `""` |
| `increase-request-cpu-algo` | `increaseRequestCpuAlgo` | Algorithm to increase CPU request. | `"ratio"`, `"margin"` | `"ratio"` |
| `increase-request-cpu-value` | `increaseRequestCpuValue` | Value used to increase CPU request. Accepts any numeric value. | Any numeric value | `"1"` |

* * *

#### 3. Memory Request Settings

| Annotation Key | ResourcesConfig Field | Description | Options | Default |
| --- | --- | --- | --- | --- |
| `request-memory-apply-mode` | `requestMemoryApplyMode` | Memory request recommendation mode. | `"enforce"`, `"off"` | `"enforce"` |
| `min-request-memory` | `minRequestMemory` | Minimum memory request value. Accepts any valid memory value (e.g., `"200Mi"`). | Any valid memory value | `""` |
| `max-request-memory` | `maxRequestMemory` | Maximum memory request value. Accepts any valid memory value (e.g., `"20Gi"`). | Any valid memory value | `""` |
| `request-memory-apply-target` | `requestMemoryApplyTarget` | Select which recommendation to apply for memory request. | `"frugal"`, `"balanced"`, `"peak"` | `"balanced"` |
| `request-memory-scale-direction` | `requestMemoryScaleDirection` | Allowed scaling direction for memory request. | `"both"`, `"up"`, `"down"` | `"both"` |
| `min-allowed-recommendation-memory` | `minAllowedRecommendationMemory` | Minimum allowed memory recommendation value. **Overrides VPA** `minAllowed.memory`. Accepts any valid memory value (e.g., `"200Mi"`). | Any valid memory value | `""` |
| `max-allowed-recommendation-memory` | `maxAllowedRecommendationMemory` | Maximum allowed memory recommendation value. **Overrides VPA** `maxAllowed.memory`. Accepts any valid memory value (e.g., `"20Gi"`). | Any valid memory value | `""` |
| `increase-request-memory-algo` | `increaseRequestMemoryAlgo` | Algorithm to increase memory request. | `"ratio"`, `"margin"` | `"ratio"` |
| `increase-request-memory-value` | `increaseRequestMemoryValue` | Value used to increase memory request. Accepts any numeric value. | Any numeric value | `"1"` |

* * *

#### 4. CPU Limit Settings

| Annotation Key | ResourcesConfig Field | Description | Options | Default |
| --- | --- | --- | --- | --- |
| `limit-apply-target` | `limitApplyTarget` | Select which recommendation to apply by default on limit. | `"auto"`, `"frugal"`, `"balanced"`, `"peak"` | `"auto"` |
| `limit-cpu-apply-mode` | `limitCpuApplyMode` | CPU limit apply mode. | `"enforce"`, `"off"` | `"enforce"` |
| `min-limit-cpu` | `minLimitCpu` | Minimum CPU limit value. Accepts any valid CPU value (e.g., `"200m"`). | Any valid CPU value | `""` |
| `max-limit-cpu` | `maxLimitCpu` | Maximum CPU limit value. Accepts any valid CPU value (e.g., `"4"`) | Any valid CPU value | `""` |
| `limit-cpu-apply-target` | `limitCpuApplyTarget` | Select which recommendation to apply for CPU limit. | `"auto"`, `"frugal"`, `"balanced"`, `"peak"` | `"auto"` |
| `limit-cpu-scale-direction` | `limitCpuScaleDirection` | Allowed scaling direction for CPU limit. | `"both"`, `"up"`, `"down"` | `"both"` |
| `limit-cpu-calculator-algo` | `limitCpuCalculatorAlgo` | CPU limit calculator algorithm. | `"ratio"`, `"margin"` | `"ratio"` |
| `limit-cpu-calculator-value` | `limitCpuCalculatorValue` | Value used by the CPU limit calculator algorithm. Accepts any numeric value. | Any numeric value | `"1"` |

* * *

#### 5. Memory Limit Settings

| Annotation Key | ResourcesConfig Field | Description | Options | Default |
| --- | --- | --- | --- | --- |
| `limit-apply-target` | `limitApplyTarget` | Select which recommendation to apply by default on limit. | `"auto"`, `"frugal"`, `"balanced"`, `"peak"` | `"auto"` |
| `limit-memory-apply-mode` | `limitMemoryApplyMode` | Memory limit apply mode. | `"enforce"`, `"off"` | `"enforce"` |
| `min-limit-memory` | `minLimitMemory` | Minimum memory limit value. Accepts any valid memory value (e.g., `"200Mi"`). | Any valid memory value | `""` |
| `max-limit-memory` | `maxLimitMemory` | Maximum memory limit value. Accepts any valid memory value (e.g., `"8Gi"`). | Any valid memory value | `""` |
| `limit-memory-apply-target` | `limitMemoryApplyTarget` | Select which recommendation to apply for memory limit. | `"auto"`, `"frugal"`, `"balanced"`, `"peak"` | `"auto"` |
| `limit-memory-scale-direction` | `limitMemoryScaleDirection` | Allowed scaling direction for memory limit. | `"both"`, `"up"`, `"down"` | `"both"` |
| `limit-memory-calculator-algo` | `limitMemoryCalculatorAlgo` | Memory limit calculator algorithm. | `"ratio"`, `"margin"` | `"ratio"` |
| `limit-memory-calculator-value` | `limitMemoryCalculatorValue` | Value used by the memory limit calculator algorithm. Accepts any numeric value. | Any numeric value | `"1"` |

* * *

#### 6. Cross-Resource Calculation

| Annotation Key | ResourcesConfig Field | Description | Options | Default |
| --- | --- | --- | --- | --- |
| `memory-request-from-cpu-enabled` | `memoryRequestFromCpuEnabled` | Calculate memory request from CPU request instead of using the recommendation. | `"true"`, `"false"` | `"false"` |
| `memory-request-from-cpu-algo` | `memoryRequestFromCpuAlgo` | Algorithm to calculate memory request from CPU. | `"ratio"`, `"margin"` | `"ratio"` |
| `memory-request-from-cpu-value` | `memoryRequestFromCpuValue` | Value used for calculating memory request from CPU. Accepts any numeric value. | Any numeric value | `"2"` |
| `memory-limit-from-cpu-enabled` | `memoryLimitFromCpuEnabled` | Calculate memory limit from CPU limit instead of using the recommendation. | `"true"`, `"false"` | `"false"` |
| `memory-limit-from-cpu-algo` | `memoryLimitFromCpuAlgo` | Algorithm to calculate memory limit from CPU. | `"ratio"`, `"margin"` | `"ratio"` |
| `memory-limit-from-cpu-value` | `memoryLimitFromCpuValue` | Value used for calculating memory limit from CPU. Accepts any numeric value. | Any numeric value | `"2"` |

* * *

#### 7. Minimum Difference Thresholds

| Annotation Key | ResourcesConfig Field | Description | Options | Default |
| --- | --- | --- | --- | --- |
| `min-diff-cpu-request-algo` | `minDiffCpuRequestAlgo` | Algorithm for minimum CPU request difference. | `"ratio"`, `"margin"` | `"ratio"` |
| `min-diff-cpu-request-value` | `minDiffCpuRequestValue` | Value for minimum CPU request difference calculation. Accepts any numeric value. | Any numeric value | `"0"` |
| `min-diff-memory-request-algo` | `minDiffMemoryRequestAlgo` | Algorithm for minimum memory request difference. | `"ratio"`, `"margin"` | `"ratio"` |
| `min-diff-memory-request-value` | `minDiffMemoryRequestValue` | Value for minimum memory request difference calculation. Accepts any numeric value. | Any numeric value | `"0"` |
| `min-diff-cpu-limit-algo` | `minDiffCpuLimitAlgo` | Algorithm for minimum CPU limit difference. | `"ratio"`, `"margin"` | `"ratio"` |
| `min-diff-cpu-limit-value` | `minDiffCpuLimitValue` | Value for minimum CPU limit difference calculation. Accepts any numeric value. | Any numeric value | `"0"` |
| `min-diff-memory-limit-algo` | `minDiffMemoryLimitAlgo` | Algorithm for minimum memory limit difference. | `"ratio"`, `"margin"` | `"ratio"` |
| `min-diff-memory-limit-value` | `minDiffMemoryLimitValue` | Value for minimum memory limit difference calculation. Accepts any numeric value. | Any numeric value | `"0"` |

#### 8. Direct Resource Specifications

| Annotation Key | ResourcesConfig Field | Description | Options | Default |
| --- | --- | --- | --- | --- |
| `request-cpu` | `requestCpu` | Direct CPU request value. Takes precedence over VPA recommendations. | Any valid CPU value | `""` |
| `request-memory` | `requestMemory` | Direct memory request value. Takes precedence over VPA recommendations. | Any valid memory value | `""` |
| `limit-cpu` | `limitCpu` | Direct CPU limit value. Takes precedence over VPA recommendations. | Any valid CPU value | `""` |
| `limit-memory` | `limitMemory` | Direct memory limit value. Takes precedence over VPA recommendations. | Any valid memory value | `""` |
| N/A | `request.cpu` | Kubernetes-native style CPU request. Takes precedence over VPA recommendations. | Any valid CPU value | `""` |
| N/A | `request.memory` | Kubernetes-native style memory request. Takes precedence over VPA recommendations. | Any valid memory value | `""` |
| N/A | `limit.cpu` | Kubernetes-native style CPU limit. Takes precedence over VPA recommendations. | Any valid CPU value | `""` |
| N/A | `limit.memory` | Kubernetes-native style memory limit. Takes precedence over VPA recommendations. | Any valid memory value | `""` |

#### 9. Container-Specific Configurations

The ResourcesConfig CRD allows you to specify container-specific configurations using the `containerConfigs` field. This is a map where the keys are container names and the values are objects containing any of the resource configuration fields, including direct resource specifications.

### Example Usage

#### Complete ResourcesConfig Example (Using VPA Recommendations):

```yaml
apiVersion: oblik.socialgouv.io/v1
kind: ResourcesConfig
metadata:
  name: web-app-resources
  namespace: default
spec:
  targetRef:
    kind: Deployment
    name: web-app
  # Basic settings
  cron: "0 3 * * *"
  cronAddRandomMax: "60m"
  dryRun: false
  webhookEnabled: true
  
  # General settings
  requestApplyTarget: "balanced"
  limitApplyTarget: "auto"
  
  # CPU request settings
  requestCpuApplyMode: "enforce"
  minRequestCpu: "100m"
  maxRequestCpu: "2"
  requestCpuApplyTarget: "balanced"
  
  # Memory request settings
  requestMemoryApplyMode: "enforce"
  minRequestMemory: "128Mi"
  maxRequestMemory: "4Gi"
  
  # CPU limit settings
  limitCpuApplyMode: "enforce"
  minLimitCpu: "200m"
  
  # Memory limit settings
  limitMemoryApplyMode: "enforce"
  minLimitMemory: "256Mi"
  
  # Container-specific settings
  containerConfigs:
    nginx:
      minRequestCpu: "50m"
      maxRequestMemory: "256Mi"
```

#### Direct Resource Specifications (Flat Style):

```yaml
apiVersion: oblik.socialgouv.io/v1
kind: ResourcesConfig
metadata:
  name: web-app-resources
  namespace: default
spec:
  targetRef:
    kind: Deployment
    name: web-app
  
  # Direct resource specifications (flat style)
  requestCpu: "100m"
  requestMemory: "128Mi"
  limitCpu: "200m"
  limitMemory: "256Mi"
  
  # Container-specific settings
  containerConfigs:
    nginx:
      requestCpu: "50m"
      requestMemory: "64Mi"
      limitCpu: "100m"
      limitMemory: "128Mi"
```

#### Direct Resource Specifications (Kubernetes-Native Style):

```yaml
apiVersion: oblik.socialgouv.io/v1
kind: ResourcesConfig
metadata:
  name: web-app-resources
  namespace: default
spec:
  targetRef:
    kind: Deployment
    name: web-app
  
  # Kubernetes-native style resource specifications
  request:
    cpu: "100m"
    memory: "128Mi"
  limit:
    cpu: "200m"
    memory: "256Mi"
  
  # Container-specific settings
  containerConfigs:
    nginx:
      request:
        cpu: "50m"
        memory: "64Mi"
      limit:
        cpu: "100m"
        memory: "128Mi"
```

#### Comparison: Annotations vs. ResourcesConfig

The same configuration using annotations would look like:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: web-app
  namespace: default
  labels:
    oblik.socialgouv.io/enabled: "true"
  annotations:
    # Basic settings
    oblik.socialgouv.io/cron: "0 3 * * *"
    oblik.socialgouv.io/cron-add-random-max: "60m"
    oblik.socialgouv.io/dry-run: "false"
    oblik.socialgouv.io/webhook-enabled: "true"
    
    # General settings
    oblik.socialgouv.io/request-apply-target: "balanced"
    oblik.socialgouv.io/limit-apply-target: "auto"
    
    # CPU request settings
    oblik.socialgouv.io/request-cpu-apply-mode: "enforce"
    oblik.socialgouv.io/min-request-cpu: "100m"
    oblik.socialgouv.io/max-request-cpu: "2"
    oblik.socialgouv.io/request-cpu-apply-target: "balanced"
    
    # Memory request settings
    oblik.socialgouv.io/request-memory-apply-mode: "enforce"
    oblik.socialgouv.io/min-request-memory: "128Mi"
    oblik.socialgouv.io/max-request-memory: "4Gi"
    
    # CPU limit settings
    oblik.socialgouv.io/limit-cpu-apply-mode: "enforce"
    oblik.socialgouv.io/min-limit-cpu: "200m"
    
    # Memory limit settings
    oblik.socialgouv.io/limit-memory-apply-mode: "enforce"
    oblik.socialgouv.io/min-limit-memory: "256Mi"
    
    # Container-specific settings
    oblik.socialgouv.io/min-request-cpu.nginx: "50m"
    oblik.socialgouv.io/max-request-memory.nginx: "256Mi"
spec:
  # ...
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

| Environment Variable | Description | Options | Default |
| --- | --- | --- | --- |
| `OBLIK_DEFAULT_CRON` | Default cron expression for scheduling when the recommendations are applied. | Any valid cron expression | `"0 2 * * *"` |
| `OBLIK_DEFAULT_CRON_ADD_RANDOM_MAX` | Maximum random delay added to the cron schedule. | Duration (e.g., `"120m"`) | `"120m"` |
| `OBLIK_DEFAULT_DRY_RUN` | If set to `"true"`, Oblik will simulate the updates without applying them. | `"true"`, `"false"` | `"false"` |
| `OBLIK_DEFAULT_WEBHOOK_ENABLED` | Enable mutating webhook resources enforcement. | `"true"`, `"false"` | `"true"` |
| `OBLIK_DEFAULT_REQUEST_CPU_APPLY_MODE` | CPU request recommendation mode. | `"enforce"`, `"off"` | `"enforce"` |
| `OBLIK_DEFAULT_REQUEST_MEMORY_APPLY_MODE` | Memory request recommendation mode. | `"enforce"`, `"off"` | `"enforce"` |
| `OBLIK_DEFAULT_LIMIT_CPU_APPLY_MODE` | CPU limit apply mode. | `"enforce"`, `"off"` | `"enforce"` |
| `OBLIK_DEFAULT_LIMIT_MEMORY_APPLY_MODE` | Memory limit apply mode. | `"enforce"`, `"off"` | `"enforce"` |
| `OBLIK_DEFAULT_LIMIT_CPU_CALCULATOR_ALGO` | Algorithm to use for calculating CPU limits. | `"ratio"`, `"margin"` | `"ratio"` |
| `OBLIK_DEFAULT_LIMIT_MEMORY_CALCULATOR_ALGO` | Algorithm to use for calculating memory limits. | `"ratio"`, `"margin"` | `"ratio"` |
| `OBLIK_DEFAULT_LIMIT_CPU_CALCULATOR_VALUE` | Value to use with the CPU limit calculator algorithm. | Any numeric value | `"1"` |
| `OBLIK_DEFAULT_LIMIT_MEMORY_CALCULATOR_VALUE` | Value to use with the memory limit calculator algorithm. | Any numeric value | `"1"` |
| `OBLIK_DEFAULT_UNPROVIDED_APPLY_DEFAULT_REQUEST_CPU` | Default behavior for CPU requests if not provided. | `"off"`, `"minAllowed"`, `"maxAllowed"`, or value (e.g., `"100m"`) | `"off"` |
| `OBLIK_DEFAULT_UNPROVIDED_APPLY_DEFAULT_REQUEST_MEMORY` | Default behavior for memory requests if not provided. | `"off"`, `"minAllowed"`, `"maxAllowed"`, or value (e.g., `"128Mi"`) | `"off"` |
| `OBLIK_DEFAULT_INCREASE_REQUEST_CPU_ALGO` | Algorithm to use for increasing CPU requests. | `"ratio"`, `"margin"` | `"ratio"` |
| `OBLIK_DEFAULT_INCREASE_REQUEST_CPU_VALUE` | Value to use with the algorithm for increasing CPU requests. | Any numeric value | `"1"` |
| `OBLIK_DEFAULT_INCREASE_REQUEST_MEMORY_ALGO` | Algorithm to use for increasing memory requests. | `"ratio"`, `"margin"` | `"ratio"` |
| `OBLIK_DEFAULT_INCREASE_REQUEST_MEMORY_VALUE` | Value to use with the algorithm for increasing memory requests. | Any numeric value | `"1"` |
| `OBLIK_DEFAULT_MIN_LIMIT_CPU` | Value used to cap minimum CPU limit. | Any valid CPU value (e.g., `"200m"`) | `""` |
| `OBLIK_DEFAULT_MAX_LIMIT_CPU` | Value used to cap maximum CPU limit. | Any valid CPU value (e.g., `"4"`) | `""` |
| `OBLIK_DEFAULT_MIN_LIMIT_MEMORY` | Value used to cap minimum memory limit. | Any valid memory value (e.g., `"200Mi"`) | `""` |
| `OBLIK_DEFAULT_MAX_LIMIT_MEMORY` | Value used to cap maximum memory limit. | Any valid memory value (e.g., `"8Gi"`) | `""` |
| `OBLIK_DEFAULT_MIN_REQUEST_CPU` | Value used to cap minimum CPU request. | Any valid CPU value (e.g., `"80m"`) | `""` |
| `OBLIK_DEFAULT_MAX_REQUEST_CPU` | Value used to cap maximum CPU request. | Any valid CPU value (e.g., `"8"`) | `""` |
| `OBLIK_DEFAULT_MIN_REQUEST_MEMORY` | Value used to cap minimum memory request. | Any valid memory value (e.g., `"200Mi"`) | `""` |
| `OBLIK_DEFAULT_MAX_REQUEST_MEMORY` | Value used to cap maximum memory request. | Any valid memory value (e.g., `"20Gi"`) | `""` |
| `OBLIK_DEFAULT_MIN_ALLOWED_RECOMMENDATION_CPU` | Minimum allowed CPU recommendation value. Overrides VPA `minAllowed.cpu`. | Any valid CPU value | `""` |
| `OBLIK_DEFAULT_MAX_ALLOWED_RECOMMENDATION_CPU` | Maximum allowed CPU recommendation value. Overrides VPA `maxAllowed.cpu`. | Any valid CPU value | `""` |
| `OBLIK_DEFAULT_MIN_ALLOWED_RECOMMENDATION_MEMORY` | Minimum allowed memory recommendation value. Overrides VPA `minAllowed.memory`. | Any valid memory value | `""` |
| `OBLIK_DEFAULT_MAX_ALLOWED_RECOMMENDATION_MEMORY` | Maximum allowed memory recommendation value. Overrides VPA `maxAllowed.memory`. | Any valid memory value | `""` |
| `OBLIK_DEFAULT_MIN_DIFF_CPU_REQUEST_ALGO` | Algorithm to calculate the minimum CPU request difference for applying recommendations. | `"ratio"`, `"margin"` | `"ratio"` |
| `OBLIK_DEFAULT_MIN_DIFF_CPU_REQUEST_VALUE` | Value used for minimum CPU request difference calculation. | Any numeric value | `"0"` |
| `OBLIK_DEFAULT_MIN_DIFF_MEMORY_REQUEST_ALGO` | Algorithm to calculate the minimum memory request difference for applying recommendations. | `"ratio"`, `"margin"` | `"ratio"` |
| `OBLIK_DEFAULT_MIN_DIFF_MEMORY_REQUEST_VALUE` | Value used for minimum memory request difference calculation. | Any numeric value | `"0"` |
| `OBLIK_DEFAULT_MIN_DIFF_CPU_LIMIT_ALGO` | Algorithm to calculate the minimum CPU limit difference for applying recommendations. | `"ratio"`, `"margin"` | `"ratio"` |
| `OBLIK_DEFAULT_MIN_DIFF_CPU_LIMIT_VALUE` | Value used for minimum CPU limit difference calculation. | Any numeric value | `"0"` |
| `OBLIK_DEFAULT_MIN_DIFF_MEMORY_LIMIT_ALGO` | Algorithm to calculate the minimum memory limit difference for applying recommendations. | `"ratio"`, `"margin"` | `"ratio"` |
| `OBLIK_DEFAULT_MIN_DIFF_MEMORY_LIMIT_VALUE` | Value used for minimum memory limit difference calculation. | Any numeric value | `"0"` |
| `OBLIK_DEFAULT_MEMORY_REQUEST_FROM_CPU_ENABLED` | Calculate memory request from CPU request instead of recommendation. | `"true"`, `"false"` | `"false"` |
| `OBLIK_DEFAULT_MEMORY_LIMIT_FROM_CPU_ENABLED` | Calculate memory limit from CPU limit instead of recommendation. | `"true"`, `"false"` | `"false"` |
| `OBLIK_DEFAULT_MEMORY_REQUEST_FROM_CPU_ALGO` | Algorithm to calculate memory request based on CPU request. | `"ratio"`, `"margin"` | `"ratio"` |
| `OBLIK_DEFAULT_MEMORY_REQUEST_FROM_CPU_VALUE` | Value used for calculating memory request from CPU request. | Any numeric value | `"2"` |
| `OBLIK_DEFAULT_MEMORY_LIMIT_FROM_CPU_ALGO` | Algorithm to calculate memory limit based on CPU limit. | `"ratio"`, `"margin"` | `"ratio"` |
| `OBLIK_DEFAULT_MEMORY_LIMIT_FROM_CPU_VALUE` | Value used for calculating memory limit from CPU limit. | Any numeric value | `"2"` |
| `OBLIK_DEFAULT_REQUEST_APPLY_TARGET` | Select which recommendation to apply by default on request. | `"frugal"`, `"balanced"`, `"peak"` | `"balanced"` |
| `OBLIK_DEFAULT_REQUEST_CPU_APPLY_TARGET` | Select which recommendation to apply for CPU request. | `"frugal"`, `"balanced"`, `"peak"` | `"balanced"` |
| `OBLIK_DEFAULT_REQUEST_MEMORY_APPLY_TARGET` | Select which recommendation to apply for memory request. | `"frugal"`, `"balanced"`, `"peak"` | `"balanced"` |
| `OBLIK_DEFAULT_LIMIT_APPLY_TARGET` | Select which recommendation to apply by default on limit. | `"auto"`, `"frugal"`, `"balanced"`, `"peak"` | `"auto"` |
| `OBLIK_DEFAULT_LIMIT_CPU_APPLY_TARGET` | Select which recommendation to apply for CPU limit. | `"auto"`, `"frugal"`, `"balanced"`, `"peak"` | `"auto"` |
| `OBLIK_DEFAULT_LIMIT_MEMORY_APPLY_TARGET` | Select which recommendation to apply for memory limit. | `"auto"`, `"frugal"`, `"balanced"`, `"peak"` | `"auto"` |
| `OBLIK_DEFAULT_REQUEST_CPU_SCALE_DIRECTION` | Allowed scaling direction for CPU request. | `"both"`, `"up"`, `"down"` | `"both"` |
| `OBLIK_DEFAULT_REQUEST_MEMORY_SCALE_DIRECTION` | Allowed scaling direction for memory request. | `"both"`, `"up"`, `"down"` | `"both"` |
| `OBLIK_DEFAULT_LIMIT_CPU_SCALE_DIRECTION` | Allowed scaling direction for CPU limit. | `"both"`, `"up"`, `"down"` | `"both"` |
| `OBLIK_DEFAULT_LIMIT_MEMORY_SCALE_DIRECTION` | Allowed scaling direction for memory limit. | `"both"`, `"up"`, `"down"` | `"both"` |
| `OBLIK_MATTERMOST_WEBHOOK_URL` | Webhook URL for Mattermost notifications. | URL | `""` |

**Notes:**

* **Algorithms:** The options `"ratio"` and `"margin"` refer to how values are calculated:
    
    * **`ratio`**: Multiplies the base value by a ratio (e.g., `value = base * ratio`).
    * **`margin`**: Adds a fixed margin to the base value (e.g., `value = base + margin`).
* **Scaling Directions:** The scaling direction options control whether resources can be increased, decreased, or both when applying recommendations:
    
    * **`both`**: Allows both scaling up and down.
    * **`up`**: Only allows scaling up (increasing resources).
    * **`down`**: Only allows scaling down (decreasing resources).
* **Apply Modes:**
    
    * **`enforce`**: Oblik will enforce the recommended values.
    * **`off`**: Oblik will not apply recommendations for this resource.
* **Apply Targets:**
    
    * **`frugal`**: Use the lower bound of recommendations.
    * **`balanced`**: Use the middle value of recommendations.
    * **`peak`**: Use the upper bound of recommendations.
    * **`auto`**: Oblik will decide the best target based on other settings.
* **Unprovided Defaults:**
    
    * **`off`**: Do not apply a default if the recommendation is missing.
    * **`minAllowed`/`maxAllowed`**: Use the VPA's `minAllowed` or `maxAllowed` values.
    * **Specific Value**: Provide a specific value to use as the default.

Feel free to set these environment variables in your operator's deployment manifest to establish cluster-wide defaults. Individual workloads can override these defaults by specifying annotations on their resource definitions.


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
