package logical

import (
	"github.com/SocialGouv/oblik/pkg/config"
	"github.com/SocialGouv/oblik/pkg/reporting"
	corev1 "k8s.io/api/core/v1"
	vpa "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
)

func UpdateContainerResources(containers []corev1.Container, vpaResource *vpa.VerticalPodAutoscaler, vcfg *config.VpaWorkloadCfg) *reporting.UpdateResult {
	requestRecommandations := getRequestTargetRecommandations(vpaResource, vcfg)
	requestRecommandations = setUnprovidedDefaultRecommandations(containers, requestRecommandations, vpaResource, vcfg)

	limitRecommandations := getLimitTargetRecommandations(vpaResource, vcfg)
	limitRecommandations = setUnprovidedDefaultRecommandations(containers, limitRecommandations, vpaResource, vcfg)

	update := applyRecommandationsToContainers(containers, requestRecommandations, limitRecommandations, vcfg)
	return update
}
