package logical

import (
	"github.com/SocialGouv/oblik/pkg/config"
	"k8s.io/apimachinery/pkg/api/resource"
	vpa "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
)

type TargetRecommandation struct {
	Cpu           *resource.Quantity
	Memory        *resource.Quantity
	ContainerName string
}

func getRequestTargetRecommandations(vpaResource *vpa.VerticalPodAutoscaler, scfg *config.StrategyConfig) []TargetRecommandation {
	recommandations := []TargetRecommandation{}
	if vpaResource.Status.Recommendation != nil {
		for _, containerRecommendation := range vpaResource.Status.Recommendation.ContainerRecommendations {
			containerName := containerRecommendation.ContainerName
			recommandation := TargetRecommandation{
				ContainerName: containerName,
			}
			switch scfg.GetRequestCpuApplyTarget(containerName) {
			case config.RequestApplyTargetFrugal:
				recommandation.Cpu = containerRecommendation.LowerBound.Cpu()
			case config.RequestApplyTargetBalanced:
				recommandation.Cpu = containerRecommendation.Target.Cpu()
			case config.RequestApplyTargetPeak:
				recommandation.Cpu = containerRecommendation.UpperBound.Cpu()
			}
			switch scfg.GetRequestMemoryApplyTarget(containerName) {
			case config.RequestApplyTargetFrugal:
				recommandation.Memory = containerRecommendation.LowerBound.Memory()
			case config.RequestApplyTargetBalanced:
				recommandation.Memory = containerRecommendation.Target.Memory()
			case config.RequestApplyTargetPeak:
				recommandation.Memory = containerRecommendation.UpperBound.Memory()
			}
			recommandations = append(recommandations, recommandation)
		}
	}
	return recommandations
}

func getLimitTargetRecommandations(vpaResource *vpa.VerticalPodAutoscaler, scfg *config.StrategyConfig) []TargetRecommandation {
	recommandations := []TargetRecommandation{}
	if vpaResource.Status.Recommendation != nil {
		for _, containerRecommendation := range vpaResource.Status.Recommendation.ContainerRecommendations {
			containerName := containerRecommendation.ContainerName
			recommandation := TargetRecommandation{
				ContainerName: containerName,
			}
			switch scfg.GetLimitCpuApplyTarget(containerName) {
			case config.LimitApplyTargetFrugal:
				recommandation.Cpu = containerRecommendation.LowerBound.Cpu()
			case config.LimitApplyTargetBalanced:
				recommandation.Cpu = containerRecommendation.Target.Cpu()
			case config.LimitApplyTargetPeak:
				recommandation.Cpu = containerRecommendation.UpperBound.Cpu()
			}
			switch scfg.GetLimitMemoryApplyTarget(containerName) {
			case config.LimitApplyTargetFrugal:
				recommandation.Memory = containerRecommendation.LowerBound.Memory()
			case config.LimitApplyTargetBalanced:
				recommandation.Memory = containerRecommendation.Target.Memory()
			case config.LimitApplyTargetPeak:
				recommandation.Memory = containerRecommendation.UpperBound.Memory()
			}
			recommandations = append(recommandations, recommandation)
		}
	}
	return recommandations
}
