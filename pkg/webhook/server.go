package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"reflect"

	"github.com/SocialGouv/oblik/pkg/client"
	"github.com/SocialGouv/oblik/pkg/config"
	"github.com/SocialGouv/oblik/pkg/logical"
	"github.com/SocialGouv/oblik/pkg/utils"
	cnpgv1 "github.com/cloudnative-pg/cloudnative-pg/api/v1"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	admissionv1 "k8s.io/api/admission/v1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/types"
	autoscalingv1 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
	"k8s.io/klog/v2"
)

var (
	Port        = 9443
	MetricsPort = 9090
	CertFile    = "/etc/webhook/certs/cert.pem"
	KeyFile     = "/etc/webhook/certs/key.pem"
)

var (
	scheme = runtime.NewScheme()
	codecs = serializer.NewCodecFactory(scheme)
)

var (
	operatorNamespace      = os.Getenv("NAMESPACE")
	operatorServiceAccount = os.Getenv("SERVICE_ACCOUNT")
)

var operatorUsername string

func init() {
	_ = admissionv1.AddToScheme(scheme)
	_ = cnpgv1.AddToScheme(scheme)
	operatorUsername = fmt.Sprintf("system:serviceaccount:%s:%s", operatorNamespace, operatorServiceAccount)
}

func Server(ctx context.Context, kubeClients *client.KubeClients) error {
	server := &http.Server{
		Addr: fmt.Sprintf(":%d", Port),
	}

	mux := http.NewServeMux()
	server.Handler = mux

	mux.HandleFunc("/healthz", HealthCheckHandler)
	mux.HandleFunc("/mutate", func(writer http.ResponseWriter, request *http.Request) {
		MutateHandler(writer, request, kubeClients)
	})

	metricsHandler := promhttp.Handler()
	go startMetricsServer(metricsHandler)
	return server.ListenAndServeTLS(CertFile, KeyFile)
}

func startMetricsServer(metricsHandler http.Handler) {
	klog.Infof("Starting metrics server on port %d\n", MetricsPort)
	metricsRouter := http.NewServeMux()
	metricsRouter.Handle("/metrics", metricsHandler)

	metricsServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", MetricsPort),
		Handler: metricsRouter,
	}

	if err := metricsServer.ListenAndServe(); err != nil {
		klog.Fatal("Failed to start metrics server:", err)
	}
}

func HealthCheckHandler(writer http.ResponseWriter, _ *http.Request) {
	writer.WriteHeader(http.StatusOK)
}

func MutateHandler(writer http.ResponseWriter, request *http.Request, kubeClients *client.KubeClients) {
	var admissionReview admissionv1.AdmissionReview

	// Read the request body
	body, err := io.ReadAll(request.Body)
	if err != nil {
		klog.Errorf("Could not read request body: %v", err)
		http.Error(writer, "could not read request", http.StatusBadRequest)
		return
	}
	defer request.Body.Close()

	// Decode the admission review request
	if _, _, err := codecs.UniversalDeserializer().Decode(body, nil, &admissionReview); err != nil {
		klog.Errorf("Could not decode request: %v", err)
		http.Error(writer, "could not decode request", http.StatusBadRequest)
		return
	}

	// Ensure that the Request field is not nil
	if admissionReview.Request == nil {
		klog.Error("AdmissionReview.Request is nil")
		http.Error(writer, "admissionReview.Request is nil", http.StatusBadRequest)
		return
	}

	// Check if the request comes from the operator's service account
	if admissionReview.Request.UserInfo.Username == operatorUsername {
		// Skip mutation as update is requested by operator
		// klog.Infof("Skipping mutation for request from operator service account: %s", operatorUsername)
		allowRequest(writer, admissionReview.Request.UID)
		return
	}

	err = MutateExec(writer, request, admissionReview, kubeClients)
	if err != nil {
		klog.Error(err)
		allowRequest(writer, admissionReview.Request.UID)
	}
}

func MutateExec(writer http.ResponseWriter, request *http.Request, admissionReview admissionv1.AdmissionReview, kubeClients *client.KubeClients) error {

	admissionRequest := admissionReview.Request

	// Parse the object and update if needed
	raw := admissionRequest.Object.Raw
	obj := &unstructured.Unstructured{}
	if err := json.Unmarshal(raw, obj); err != nil {
		return fmt.Errorf("Could not unmarshal object: %v", err)
	}

	var configurable *config.Configurable
	var containers []corev1.Container
	// Determine the kind of the object and convert it to the respective type
	switch obj.GetKind() {
	case "Deployment":
		deployment := &appsv1.Deployment{}
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, deployment); err != nil {
			return fmt.Errorf("Could not convert to Deployment: %v", err)
		}
		configurable = config.CreateConfigurable(deployment)
		containers = deployment.Spec.Template.Spec.Containers
	case "StatefulSet":
		statefulSet := &appsv1.StatefulSet{}
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, statefulSet); err != nil {
			return fmt.Errorf("Could not convert to StatefulSet: %v", err)
		}
		configurable = config.CreateConfigurable(statefulSet)
		containers = statefulSet.Spec.Template.Spec.Containers
	case "DaemonSet":
		daemonSet := &appsv1.DaemonSet{}
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, daemonSet); err != nil {
			return fmt.Errorf("Could not convert to DaemonSet: %v", err)
		}
		configurable = config.CreateConfigurable(daemonSet)
		containers = daemonSet.Spec.Template.Spec.Containers
	case "CronJob":
		cronJob := &batchv1.CronJob{}
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, cronJob); err != nil {
			return fmt.Errorf("Could not convert to CronJob: %v", err)
		}
		configurable = config.CreateConfigurable(cronJob)
		containers = cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers
	case "Cluster":
		if obj.GetAPIVersion() != "postgresql.cnpg.io/v1" {
			return fmt.Errorf("Unsupported Cluster kind from apiVersion: %s", obj.GetAPIVersion())
		}
		cnpgCluster := &cnpgv1.Cluster{}
		_, _, err := codecs.UniversalDeserializer().Decode(admissionRequest.Object.Raw, nil, cnpgCluster)
		if err != nil {
			return fmt.Errorf("Could not decode to Cluster: %v", err)
		}
		configurable = config.CreateConfigurable(cnpgCluster)
		containers = []corev1.Container{
			{
				Name:      "postgres",
				Resources: cnpgCluster.Spec.Resources,
			},
		}
	default:
		return fmt.Errorf("Unsupported kind: %v", obj.GetKind())
	}

	scfg := config.CreateStrategyConfig(configurable)
	if !scfg.WebhookEnabled || !scfg.Enabled {
		allowRequest(writer, admissionReview.Request.UID)
		return nil
	}

	vpaResource := getVPAResource(obj, kubeClients)
	var requestRecommendations, limitRecommendations []logical.TargetRecommendation
	if vpaResource != nil {
		requestRecommendations = logical.GetRequestTargetRecommendations(vpaResource, scfg)
		limitRecommendations = logical.GetLimitTargetRecommendations(vpaResource, scfg)
	} else {
		requestRecommendations = []logical.TargetRecommendation{}
		limitRecommendations = []logical.TargetRecommendation{}
	}

	requestRecommendations = logical.SetUnprovidedDefaultRecommendations(containers, requestRecommendations, scfg, nil)
	limitRecommendations = logical.SetUnprovidedDefaultRecommendations(containers, limitRecommendations, scfg, nil)
	logical.ApplyRecommendationsToContainers(containers, requestRecommendations, limitRecommendations, scfg)

	switch obj.GetKind() {
	case "Deployment":
		deployment := configurable.Get().(*appsv1.Deployment)
		deployment.Spec.Template.Spec.Containers = containers
		updated, err := runtime.DefaultUnstructuredConverter.ToUnstructured(deployment)
		if err != nil {
			return fmt.Errorf("Could not convert updated Deployment back to unstructured: %v", err)
		}
		obj.Object = updated
	case "StatefulSet":
		statefulSet := configurable.Get().(*appsv1.StatefulSet)
		statefulSet.Spec.Template.Spec.Containers = containers
		updated, err := runtime.DefaultUnstructuredConverter.ToUnstructured(statefulSet)
		if err != nil {
			return fmt.Errorf("Could not convert updated StatefulSet back to unstructured: %v", err)
		}
		obj.Object = updated
	case "DaemonSet":
		daemonSet := configurable.Get().(*appsv1.DaemonSet)
		daemonSet.Spec.Template.Spec.Containers = containers
		updated, err := runtime.DefaultUnstructuredConverter.ToUnstructured(daemonSet)
		if err != nil {
			return fmt.Errorf("Could not convert updated DaemonSet back to unstructured: %v", err)
		}
		obj.Object = updated
	case "CronJob":
		cronJob := configurable.Get().(*batchv1.CronJob)
		cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers = containers
		updated, err := runtime.DefaultUnstructuredConverter.ToUnstructured(cronJob)
		if err != nil {
			return fmt.Errorf("Could not convert updated CronJob back to unstructured: %v", err)
		}
		obj.Object = updated
	case "Cluster":
		cnpgCluster := configurable.Get().(*cnpgv1.Cluster)
		cnpgCluster.Spec.Resources = containers[0].Resources
		updated, err := runtime.DefaultUnstructuredConverter.ToUnstructured(cnpgCluster)
		if err != nil {
			return fmt.Errorf("Could not convert updated Cluster back to unstructured: %v", err)
		}
		obj.Object = updated
	}

	// Create a JSON patch
	patch, err := createJSONPatch(admissionRequest.Object.Raw, obj)
	if err != nil {
		return fmt.Errorf("Could not create JSON patch: %v", err)
	}

	// Log the created patch for debugging
	klog.Infof("Created patch: %s", string(patch))

	admissionResponse := &admissionv1.AdmissionResponse{
		UID:     admissionRequest.UID,
		Allowed: true,
		Patch:   patch, // Directly pass the raw JSON patch
		PatchType: func() *admissionv1.PatchType {
			pt := admissionv1.PatchTypeJSONPatch
			return &pt
		}(),
	}

	responseAdmissionReview := admissionv1.AdmissionReview{
		TypeMeta: admissionReview.TypeMeta,
		Response: admissionResponse,
	}

	respBytes, err := json.Marshal(responseAdmissionReview)
	if err != nil {
		return fmt.Errorf("Could not marshal response: %v", err)
	}

	writer.Header().Set("Content-Type", "application/json")
	if _, err := writer.Write(respBytes); err != nil {
		return fmt.Errorf("Could not write response: %v", err)
	}

	return nil
}

func allowRequest(writer http.ResponseWriter, uid types.UID) {
	admissionResponse := &admissionv1.AdmissionResponse{
		UID:     uid,
		Allowed: true,
	}

	admissionReview := admissionv1.AdmissionReview{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "admission.k8s.io/v1",
			Kind:       "AdmissionReview",
		},
		Response: admissionResponse,
	}

	respBytes, err := json.Marshal(admissionReview)
	if err != nil {
		klog.Errorf("Could not marshal allow response: %v", err)
		http.Error(writer, "could not marshal allow response", http.StatusInternalServerError)
		return
	}

	writer.Header().Set("Content-Type", "application/json")
	if _, err := writer.Write(respBytes); err != nil {
		klog.Errorf("Could not write allow response: %v", err)
		http.Error(writer, "could not write allow response", http.StatusInternalServerError)
		return
	}
}

func createJSONPatch(originalJSON []byte, modified *unstructured.Unstructured) ([]byte, error) {
	var original map[string]interface{}
	if err := json.Unmarshal(originalJSON, &original); err != nil {
		return nil, fmt.Errorf("could not unmarshal original object: %v", err)
	}

	patch := []map[string]interface{}{}

	// Compare and create patch operations
	compareMaps("", original, modified.Object, &patch)

	// If no changes are needed, return an empty patch
	if len(patch) == 0 {
		return []byte("[]"), nil
	}

	return json.Marshal(patch)
}

func compareMaps(prefix string, original, modified map[string]interface{}, patch *[]map[string]interface{}) {
	for key, modifiedValue := range modified {
		path := utils.GetJSONPath(prefix, key)
		originalValue, exists := original[key]

		if !exists {
			// Add operation for new fields
			*patch = append(*patch, map[string]interface{}{
				"op":    "add",
				"path":  path,
				"value": modifiedValue,
			})
		} else if !reflect.DeepEqual(originalValue, modifiedValue) {
			// Replace operation for changed fields
			switch modifiedValue.(type) {
			case map[string]interface{}:
				// Recursively compare nested maps
				if originalMap, ok := originalValue.(map[string]interface{}); ok {
					compareMaps(path, originalMap, modifiedValue.(map[string]interface{}), patch)
				} else {
					// If types don't match, replace the entire value
					*patch = append(*patch, map[string]interface{}{
						"op":    "replace",
						"path":  path,
						"value": modifiedValue,
					})
				}
			default:
				// For non-map types, use replace operation
				*patch = append(*patch, map[string]interface{}{
					"op":    "replace",
					"path":  path,
					"value": modifiedValue,
				})
			}
		}
	}

	// Check for removed fields
	for key := range original {
		if _, exists := modified[key]; !exists {
			*patch = append(*patch, map[string]interface{}{
				"op":   "remove",
				"path": utils.GetJSONPath(prefix, key),
			})
		}
	}
}

func getVPAResource(obj *unstructured.Unstructured, kubeClients *client.KubeClients) *autoscalingv1.VerticalPodAutoscaler {
	namespace := obj.GetNamespace()
	vpaName := obj.GetName()
	vpa, err := kubeClients.VpaClientset.AutoscalingV1().VerticalPodAutoscalers(namespace).Get(context.TODO(), vpaName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			klog.Infof("No VPA resource found for %s/%s: %v", namespace, vpaName, err)
		} else {
			klog.Errorf("Error retrieving VPA for %s/%s: %v", namespace, vpaName, err)
		}
		return nil
	}
	return vpa
}
