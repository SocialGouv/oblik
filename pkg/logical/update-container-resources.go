package logical

import (
	"github.com/SocialGouv/oblik/pkg/config"
	"github.com/SocialGouv/oblik/pkg/reporting"
	corev1 "k8s.io/api/core/v1"
	vpa "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
)

func UpdateContainerResources(containers []corev1.Container, vpaResource *vpa.VerticalPodAutoscaler, scfg *config.StrategyConfig) *reporting.UpdateResult {
	requestRecommandations := getRequestTargetRecommandations(vpaResource, scfg)
	requestRecommandations = setUnprovidedDefaultRecommandations(containers, requestRecommandations, vpaResource, scfg)

	limitRecommandations := getLimitTargetRecommandations(vpaResource, scfg)
	limitRecommandations = setUnprovidedDefaultRecommandations(containers, limitRecommandations, vpaResource, scfg)

	update := ApplyRecommandationsToContainers(containers, requestRecommandations, limitRecommandations, scfg)
	return update
}
