package logical

import (
	"github.com/SocialGouv/oblik/pkg/config"
	"github.com/SocialGouv/oblik/pkg/reporting"
	corev1 "k8s.io/api/core/v1"
	vpa "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
)

func UpdateContainerResources(containers []corev1.Container, vpaResource *vpa.VerticalPodAutoscaler, scfg *config.StrategyConfig) *reporting.UpdateResult {
	requestRecommandations := getRequestTargetRecommandations(vpaResource, scfg)
	requestRecommandations = SetUnprovidedDefaultRecommandations(containers, requestRecommandations, scfg, vpaResource)

	limitRecommandations := getLimitTargetRecommandations(vpaResource, scfg)
	limitRecommandations = SetUnprovidedDefaultRecommandations(containers, limitRecommandations, scfg, vpaResource)

	update := ApplyRecommandationsToContainers(containers, requestRecommandations, limitRecommandations, scfg)
	return update
}
