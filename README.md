# oblik

The Oblik Kubernetes VPA (Vertical Pod Autoscaler) Operator is designed to watch VPA objects and apply resource recommendations to Deployments and StatefulSets based on configurable annotations. This operator runs in a cron-like schedule with optional random delays to stagger updates.

## Installation

oblik can be easily deployed using the provided Helm chart:

```shell
git clone https://github.com/SocialGouv/oblik.git
cd oblik/charts/oblik
helm upgrade --install oblik . --namespace oblik
```

## Features

- Watch VPA objects and apply recommendations.
- Configurable via annotations on VPA objects.
- Supports CPU and memory resource recommendations.
- Allows random delays to stagger updates.
- Default resource requests for CPU and memory if not provided.
- Support adjustements on kinds:
    - Deployments
    - StatefulSets
    - CronJob
    - postgresql.cnpg.io/Cluster (to make the VPA work on cnpg cluster see https://github.com/cloudnative-pg/cloudnative-pg/issues/2574#issuecomment-2155389267)
- Customizable algorithms and values for increasing resource requests.
- Mattermost webhook notification on resources updates


## Configuration

To enable the Oblik operator on a VPA you need this **label** on your VPA object:

- **`oblik.socialgouv.io/enabled`**: `"true"` or `"false"`. `"false"` by default, put it `"true"` to enable oblik on the VPA.

The operator uses **annotations** on VPA objects to configure its behavior. Below are the supported annotations:

- **`oblik.socialgouv.io/cron`**: Cron expression to schedule when the recommendations are applied. (default: `"0 2 * * *"`)
- **`oblik.socialgouv.io/cron-add-random-max`**: Maximum random delay added to the cron schedule. (default: `"120m"`)
- **`oblik.socialgouv.io/request-cpu-apply-mode`**: CPU recommendation mode. Options: `enforce` (default), `off`.
- **`oblik.socialgouv.io/request-memory-apply-mode`**: Memory recommendation mode. Options: `enforce`  (default), `off`.
- **`oblik.socialgouv.io/limit-memory-apply-mode`**: Memory limit apply mode. Options: `enforce`  (default), `off`.
- **`oblik.socialgouv.io/limit-cpu-apply-mode`**: CPU limit apply mode. Options: `enforce`  (default), `off`.
- **`oblik.socialgouv.io/limit-cpu-calculator-algo`**: CPU limit calculator algorithm. Options: `ratio`  (default), `margin`.
- **`oblik.socialgouv.io/limit-memory-calculator-algo`**: Memory limit calculator algorithm. Options: `ratio`  (default), `margin`.
- **`oblik.socialgouv.io/limit-memory-calculator-value`**: Value used by the memory calculator algorithm.   Default is `1`.
- **`oblik.socialgouv.io/limit-cpu-calculator-value`**: Value used by the CPU calculator algorithm.   Default is `1`.
- **`oblik.socialgouv.io/unprovided-apply-default-request-cpu`**: Default CPU request if not provided by the VPA. Options: `off` (default), `minAllowed`, `maxAllowed`, or an arbitrary value.
- **`oblik.socialgouv.io/unprovided-apply-default-request-memory`**: Default memory request if not provided by the VPA. Options: `off` (default), `minAllowed`, `maxAllowed`, or an arbitrary value.
- **`oblik.socialgouv.io/increase-request-cpu-algo`**: Algorithm to increase CPU request. Options: `ratio` (default), `margin`.
- **`oblik.socialgouv.io/increase-request-memory-algo`**: Algorithm to increase memory request. Options: `ratio` (default), `margin`.
- **`oblik.socialgouv.io/increase-request-cpu-value`**: Value used to increase CPU request. Default is `1`.
- **`oblik.socialgouv.io/increase-request-memory-value`**: Value used to increase memory request. Default is `1`.
- **`oblik.socialgouv.io/min-limit-cpu`**: Value used to cap minimum CPU limit.
- **`oblik.socialgouv.io/max-limit-cpu`**: Value used to cap maximum CPU limit.
- **`oblik.socialgouv.io/min-limit-memory`**: Value used to cap minimum memory limit.
- **`oblik.socialgouv.io/max-limit-memory`**: Value used to cap maximum memory limit.
- **`oblik.socialgouv.io/min-request-cpu`**: Value used to cap minimum CPU request (this is like an overriding for native VPA minAllowed.cpu).
- **`oblik.socialgouv.io/max-request-cpu`**: Value used to cap maximum CPU request (this is like an overriding for native VPA maxAllowed.cpu).
- **`oblik.socialgouv.io/min-request-memory`**: Value used to cap minimum memory request. (this is like an overriding for native VPA minAllowed.memory)
- **`oblik.socialgouv.io/max-request-memory`**: Value used to cap maximum memory request. (this is like an overriding for native VPA maxAllowed.memory)
- **`oblik.socialgouv.io/min-diff-cpu-request-algo`**: Algorithm to calculate the minimum cpu request diff between actual and recommendation from which oblik will enforce recommendentation. Options: `ratio` (default), `margin`.
- **`oblik.socialgouv.io/min-diff-cpu-request-value`**: Value used to calculate the minimum cpu request diff between actual and recommendation from which oblik will enforce recommendentation. Default is `0`.
- **`oblik.socialgouv.io/min-diff-memory-request-algo`**: Algorithm to calculate the minimum memory request diff between actual and recommendation from which oblik will enforce recommendentation. Options: `ratio` (default), `margin`.
- **`oblik.socialgouv.io/min-diff-memory-request-value`**: Value used to calculate the minimum memory request diff between actual and recommendation from which oblik will enforce recommendentation. Default is `0`.
- **`oblik.socialgouv.io/min-diff-cpu-limit-algo`**: Algorithm to calculate the minimum cpu limit diff between actual and recommendation from which oblik will enforce recommendentation. Options: `ratio` (default), `margin`.
- **`oblik.socialgouv.io/min-diff-cpu-limit-value`**: Value used to calculate the minimum cpu limit diff between actual and recommendation from which oblik will enforce recommendentation. Default is `0`.
- **`oblik.socialgouv.io/min-diff-memory-limit-algo`**: Algorithm to calculate the minimum memory limit diff between actual and recommendation from which oblik will enforce recommendentation. Options: `ratio` (default), `margin`.
- **`oblik.socialgouv.io/min-diff-memory-limit-value`**: Value used to calculate the minimum memory limit diff between actual and recommendation from which oblik will enforce recommendentation. Default is `0`.


To target specific container, suffix the config annotation with name of the container, eg:
- **`oblik.socialgouv.io/min-limit-memory.hasura`**: Value used to cap minimum memory limit of container hasura.


## Usage

1. Create a VPA object with the desired annotations:

    ```yaml
    apiVersion: autoscaling.k8s.io/v1
    kind: VerticalPodAutoscaler
    metadata:
      name: example-vpa
      namespace: default
      annotations:
        oblik.socialgouv.io/cron: "0 2 * * *"
        oblik.socialgouv.io/cron-add-random-max: "120m"
        oblik.socialgouv.io/request-cpu-apply-mode: "enforce"
        oblik.socialgouv.io/request-memory-apply-mode: "enforce"
        oblik.socialgouv.io/limit-memory-apply-mode: "off"
        oblik.socialgouv.io/limit-cpu-apply-mode: "enforce"
        oblik.socialgouv.io/limit-cpu-calculator-algo: "ratio"
        oblik.socialgouv.io/limit-memory-calculator-algo: "ratio"
        oblik.socialgouv.io/limit-memory-calculator-value: "1"
        oblik.socialgouv.io/limit-cpu-calculator-value: "1"
        oblik.socialgouv.io/unprovided-apply-default-request-cpu: "100m"
        oblik.socialgouv.io/unprovided-apply-default-request-memory: "128Mi"
        oblik.socialgouv.io/increase-request-cpu-algo: "ratio"
        oblik.socialgouv.io/increase-request-memory-algo: "ratio"
        oblik.socialgouv.io/increase-request-cpu-value: "1"
        oblik.socialgouv.io/increase-request-memory-value: "1"
        oblik.socialgouv.io/min-limit-cpu: "200m"
        oblik.socialgouv.io/max-limit-cpu: "4"
        oblik.socialgouv.io/min-limit-memory: "200Mi"
        oblik.socialgouv.io/max-limit-memory: "8Gi"
        oblik.socialgouv.io/min-request-cpu: "80m"
        oblik.socialgouv.io/max-request-cpu: "8"
        oblik.socialgouv.io/min-request-memory: "200Mi"
        oblik.socialgouv.io/max-request-memory: "20Gi"
    spec:
      targetRef:
        apiVersion: "apps/v1"
        kind:       "Deployment"
        name:       "example-deployment"
      updatePolicy:
        updateMode: "Off"
    ```

2. Apply the VPA object:

    ```sh
    kubectl apply -f example-vpa.yaml
    ```

3. The operator will watch for changes and apply recommendations according to the configured cron schedule and annotations.


### Annotations Usage

#### Default CPU Request

- **`oblik.socialgouv.io/unprovided-apply-default-request-cpu`**: Specifies the default CPU request to apply if the CPU request is not provided in the resource specifications.
  - **Options**:
    - `off` (default): Do not apply any default CPU request.
    - `minAllow`: Apply the minimum allowed CPU request value.
    - `maxAllow`: Apply the maximum allowed CPU request value.
    - An arbitrary value (e.g., `"100m"`): Apply the specified CPU request value.

#### Default Memory Request

- **`oblik.socialgouv.io/unprovided-apply-default-request-memory`**: Specifies the default memory request to apply if the memory request is not provided in the resource specifications.
  - **Options**:
    - `off` (default): Do not apply any default memory request.
    - `minAllow`: Apply the minimum allowed memory request value.
    - `maxAllow`: Apply the maximum allowed memory request value.
    - An arbitrary value (e.g., `"128Mi"`): Apply the specified memory request value.


#### Increase CPU Request

- **`oblik.socialgouv.io/increase-request-cpu-algo`**: Specifies the algorithm to use for increasing CPU request.
  - **Options**:
    - `ratio` (default): Increase the CPU request by a ratio.
    - `margin`: Increase the CPU request by a fixed margin.

- **`oblik.socialgouv.io/increase-request-cpu-value`**: Specifies the value to use with the algorithm for increasing CPU request. Default is `1`.

#### Increase Memory Request

- **`oblik.socialgouv.io/increase-request-memory-algo`**: Specifies the algorithm to use for increasing memory request.
  - **Options**:
    - `ratio` (default): Increase the memory request by a ratio.
    - `margin`: Increase the memory request by a fixed margin.

- **`oblik.socialgouv.io/increase-request-memory-value`**: Specifies the value to use with the algorithm for increasing memory request. Default is `1`.


### Environment Variables

The Oblik Kubernetes VPA Operator uses the following environment variables for configuration. These environment variables allow you to set default values and customize the behavior of the operator.

* **`OBLIK_DEFAULT_CRON`**: Default cron expression for scheduling when the recommendations are applied.
    
    * **Default**: `"0 2 * * *"`
* **`OBLIK_DEFAULT_CRON_ADD_RANDOM_MAX`**: Maximum random delay added to the cron schedule.
    
    * **Default**: `"120m"`
* **`OBLIK_DEFAULT_LIMIT_CPU_CALCULATOR_ALGO`**: Default algorithm to use for calculating CPU limits.
    
    * **Options**: `ratio`, `margin`
    * **Default**: `"ratio"`
* **`OBLIK_DEFAULT_LIMIT_MEMORY_CALCULATOR_ALGO`**: Default algorithm to use for calculating memory limits.
    
    * **Options**: `ratio`, `margin`
    * **Default**: `"ratio"`
* **`OBLIK_DEFAULT_LIMIT_CPU_CALCULATOR_VALUE`**: Default value to use with the CPU limit calculator algorithm.
    
    * **Default**: `"1"`
* **`OBLIK_DEFAULT_LIMIT_MEMORY_CALCULATOR_VALUE`**: Default value to use with the memory limit calculator algorithm.
    
    * **Default**: `"1"`
* **`OBLIK_DEFAULT_UNPROVIDED_APPLY_DEFAULT_REQUEST_CPU`**: Default behavior for CPU requests if not provided.
    
    * **Options**: `off`, `minAllow`/`min`, `maxAllow`/`max`, or an arbitrary value
    * **Default**: `"off"`
* **`OBLIK_DEFAULT_UNPROVIDED_APPLY_DEFAULT_REQUEST_MEMORY`**: Default behavior for memory requests if not provided.
    
    * **Options**: `off`, `minAllow`/`min`, `maxAllow`/`max`, or an arbitrary value
    * **Default**: `"off"`
* **`OBLIK_DEFAULT_INCREASE_REQUEST_CPU_ALGO`**: Default algorithm to use for increasing CPU requests.
    
    * **Options**: `ratio`, `margin`
    * **Default**: `"ratio"`
* **`OBLIK_DEFAULT_INCREASE_REQUEST_MEMORY_ALGO`**: Default algorithm to use for increasing memory requests.
    
    * **Options**: `ratio`, `margin`
    * **Default**: `"ratio"`
* **`OBLIK_DEFAULT_INCREASE_REQUEST_CPU_VALUE`**: Default value to use with the algorithm for increasing CPU requests.
    
    * **Default**: `"1"`
* **`OBLIK_DEFAULT_INCREASE_REQUEST_MEMORY_VALUE`**: Default value to use with the algorithm for increasing memory requests.
    
    * **Default**: `"1"`

* **`OBLIK_DEFAULT_MIN_LIMIT_CPU`**: Value used to cap minimum CPU limit.
    
    * **Default**: `""`
* **`OBLIK_DEFAULT_MAX_LIMIT_CPU`**: Value used to cap maximum CPU limit.
    
    * **Default**: `""`
* **`OBLIK_DEFAULT_MIN_LIMIT_MEMORY`**: Value used to cap minimum memory limit.
    
    * **Default**: `""`
* **`OBLIK_DEFAULT_MAX_LIMIT_MEMORY`**: Value used to cap maximum memory limit.
    
    * **Default**: `""`
* **`OBLIK_DEFAULT_MIN_REQUEST_CPU`**: Value used to cap minimum CPU request.
    
    * **Default**: `""`
* **`OBLIK_DEFAULT_MAX_REQUEST_CPU`**: Value used to cap maximum CPU request.
    
    * **Default**: `""`
* **`OBLIK_DEFAULT_MIN_REQUEST_MEMORY`**: Value used to cap minimum memory request.
    
    * **Default**: `""`
* **`OBLIK_DEFAULT_MAX_REQUEST_MEMORY`**: Value used to cap maximum memory request.
    
    * **Default**: `""`

* **`OBLIK_MATTERMOST_WEBHOOK_URL`**: Webhook URL for Mattermost notifications.
    
    * **Default**: `""`



## Contributing

We welcome contributions! Please feel free to submit pull requests or open issues on our GitHub repository.

## License

This project is licensed under the MIT License - see the LICENSE file for details.