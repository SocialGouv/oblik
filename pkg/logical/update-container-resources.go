package logical

import (
	"github.com/SocialGouv/oblik/pkg/config"
	"github.com/SocialGouv/oblik/pkg/reporting"
	corev1 "k8s.io/api/core/v1"
	vpa "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
)

func UpdateContainerResources(containers []corev1.Container, vpaResource *vpa.VerticalPodAutoscaler, scfg *config.StrategyConfig) *reporting.UpdateResult {
	requestRecommendations := getRequestTargetRecommendations(vpaResource, scfg)
	requestRecommendations = SetUnprovidedDefaultRecommendations(containers, requestRecommendations, scfg, vpaResource)

	limitRecommendations := getLimitTargetRecommendations(vpaResource, scfg)
	limitRecommendations = SetUnprovidedDefaultRecommendations(containers, limitRecommendations, scfg, vpaResource)

	update := ApplyRecommendationsToContainers(containers, requestRecommendations, limitRecommendations, scfg)
	return update
}
