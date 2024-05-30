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


## Configuration

The operator uses annotations on VPA objects to configure its behavior. Below are the supported annotations:

- **`oblik.socialgouv.io/cron`**: Cron expression to schedule when the recommendations are applied. (default: `"0 2 * * *"`)
- **`oblik.socialgouv.io/cron-add-random-max`**: Maximum random delay added to the cron schedule. (default: `"120m"`)
- **`oblik.socialgouv.io/request-cpu-apply-mode`**: CPU recommendation mode. Options: `enforce`, `default`, `off`.
- **`oblik.socialgouv.io/request-memory-apply-mode`**: Memory recommendation mode. Options: `enforce`, `default`, `off`.
- **`oblik.socialgouv.io/limit-memory-apply-mode`**: Memory limit apply mode. Options: `enforce`, `default`, `off`.
- **`oblik.socialgouv.io/limit-cpu-apply-mode`**: CPU limit apply mode. Options: `enforce`, `default`, `off`.
- **`oblik.socialgouv.io/limit-cpu-calculator-algo`**: CPU limit calculator algorithm. Options: `ratio`, `margin`.
- **`oblik.socialgouv.io/limit-memory-calculator-algo`**: Memory limit calculator algorithm. Options: `ratio`, `margin`.
- **`oblik.socialgouv.io/limit-memory-calculator-value`**: Value used by the memory calculator algorithm.
- **`oblik.socialgouv.io/limit-cpu-calculator-value`**: Value used by the CPU calculator algorithm.

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


## Contributing

We welcome contributions! Please feel free to submit pull requests or open issues on our GitHub repository.

## License

This project is licensed under the MIT License - see the LICENSE file for details.