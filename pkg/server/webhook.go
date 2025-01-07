package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/SocialGouv/oblik/pkg/client"
	"github.com/SocialGouv/oblik/pkg/config"
	"github.com/SocialGouv/oblik/pkg/logical"
	cnpgv1 "github.com/cloudnative-pg/cloudnative-pg/api/v1"
	admissionv1 "k8s.io/api/admission/v1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
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

func MutateHandler(writer http.ResponseWriter, request *http.Request, kubeClients *client.KubeClients) {
	klog.V(2).Infof("Received mutation request: Method=%s, URL=%s", request.Method, request.URL)

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

	klog.V(2).Infof("Processing admission request for: Namespace=%s, Name=%s, Operation=%s",
		admissionReview.Request.Namespace,
		admissionReview.Request.Name,
		admissionReview.Request.Operation)

	// Check if the request comes from the operator's service account
	if admissionReview.Request.UserInfo.Username == operatorUsername {
		klog.V(2).Infof("Skipping mutation for request from operator service account: %s", operatorUsername)
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

	klog.V(2).Infof("Processing object: Kind=%s, Name=%s, Namespace=%s",
		obj.GetKind(),
		obj.GetName(),
		obj.GetNamespace())

	var configurable *config.Configurable
	var containers []corev1.Container
	// Determine the kind of the object and convert it to the respective type
	switch obj.GetKind() {
	case "Deployment":
		deployment := &appsv1.Deployment{}
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, deployment); err != nil {
			return fmt.Errorf("Could not convert to Deployment: %v", err)
		}
		klog.V(2).Infof("Processing Deployment: %s, Replicas=%d", deployment.Name, *deployment.Spec.Replicas)
		configurable = config.CreateConfigurable(deployment)
		containers = deployment.Spec.Template.Spec.Containers
	case "StatefulSet":
		statefulSet := &appsv1.StatefulSet{}
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, statefulSet); err != nil {
			return fmt.Errorf("Could not convert to StatefulSet: %v", err)
		}
		klog.V(2).Infof("Processing StatefulSet: %s, Replicas=%d", statefulSet.Name, *statefulSet.Spec.Replicas)
		configurable = config.CreateConfigurable(statefulSet)
		containers = statefulSet.Spec.Template.Spec.Containers
	case "DaemonSet":
		daemonSet := &appsv1.DaemonSet{}
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, daemonSet); err != nil {
			return fmt.Errorf("Could not convert to DaemonSet: %v", err)
		}
		klog.V(2).Infof("Processing DaemonSet: %s", daemonSet.Name)
		configurable = config.CreateConfigurable(daemonSet)
		containers = daemonSet.Spec.Template.Spec.Containers
	case "CronJob":
		cronJob := &batchv1.CronJob{}
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, cronJob); err != nil {
			return fmt.Errorf("Could not convert to CronJob: %v", err)
		}
		klog.V(2).Infof("Processing CronJob: %s, Schedule=%s", cronJob.Name, cronJob.Spec.Schedule)
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
		klog.V(2).Infof("Processing CNPG Cluster: %s", cnpgCluster.Name)
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
		klog.V(2).Infof("Skipping mutation: WebhookEnabled=%v, Enabled=%v", scfg.WebhookEnabled, scfg.Enabled)
		allowRequest(writer, admissionReview.Request.UID)
		return nil
	}

	vpaResource := getVPAResource(obj, kubeClients)
	klog.V(2).Infof("VPA resource found: %v", vpaResource != nil)

	var requestRecommendations, limitRecommendations []logical.TargetRecommendation
	if vpaResource != nil {
		requestRecommendations = logical.GetRequestTargetRecommendations(vpaResource, scfg)
		limitRecommendations = logical.GetLimitTargetRecommendations(vpaResource, scfg)
		klog.V(2).Infof("Got recommendations - Requests: %d, Limits: %d",
			len(requestRecommendations),
			len(limitRecommendations))
	} else {
		requestRecommendations = []logical.TargetRecommendation{}
		limitRecommendations = []logical.TargetRecommendation{}
	}

	requestRecommendations = logical.SetUnprovidedDefaultRecommendations(containers, requestRecommendations, scfg, nil)
	limitRecommendations = logical.SetUnprovidedDefaultRecommendations(containers, limitRecommendations, scfg, nil)
	klog.V(2).Infof("Final recommendations after defaults - Requests: %d, Limits: %d",
		len(requestRecommendations),
		len(limitRecommendations))

	logical.ApplyRecommendationsToContainers(containers, requestRecommendations, limitRecommendations, scfg)
	klog.V(2).Info("Applied recommendations to containers")

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

	klog.V(2).Infof("Created JSON patch of length: %d bytes", len(patch))
	klog.V(3).Infof("Patch content: %s", string(patch))

	admissionResponse := &admissionv1.AdmissionResponse{
		UID:     admissionRequest.UID,
		Allowed: true,
		Patch:   patch,
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

	klog.V(2).Infof("Successfully processed mutation request for %s/%s", obj.GetNamespace(), obj.GetName())
	return nil
}

func allowRequest(writer http.ResponseWriter, uid types.UID) {
	klog.V(2).Infof("Allowing request without mutation: UID=%s", uid)

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
