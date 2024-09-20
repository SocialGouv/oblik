package logical

import (
	"github.com/SocialGouv/oblik/pkg/config"
	"k8s.io/apimachinery/pkg/api/resource"
	vpa "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
)

type TargetRecommendation struct {
	Cpu           *resource.Quantity
	Memory        *resource.Quantity
	ContainerName string
}

func getRequestTargetRecommendations(vpaResource *vpa.VerticalPodAutoscaler, scfg *config.StrategyConfig) []TargetRecommendation {
	recommendations := []TargetRecommendation{}
	if vpaResource.Status.Recommendation != nil {
		for _, containerRecommendation := range vpaResource.Status.Recommendation.ContainerRecommendations {
			containerName := containerRecommendation.ContainerName
			recommendation := TargetRecommendation{
				ContainerName: containerName,
			}
			switch scfg.GetRequestCpuApplyTarget(containerName) {
			case config.RequestApplyTargetFrugal:
				recommendation.Cpu = containerRecommendation.LowerBound.Cpu()
			case config.RequestApplyTargetBalanced:
				recommendation.Cpu = containerRecommendation.Target.Cpu()
			case config.RequestApplyTargetPeak:
				recommendation.Cpu = containerRecommendation.UpperBound.Cpu()
			}
			switch scfg.GetRequestMemoryApplyTarget(containerName) {
			case config.RequestApplyTargetFrugal:
				recommendation.Memory = containerRecommendation.LowerBound.Memory()
			case config.RequestApplyTargetBalanced:
				recommendation.Memory = containerRecommendation.Target.Memory()
			case config.RequestApplyTargetPeak:
				recommendation.Memory = containerRecommendation.UpperBound.Memory()
			}
			recommendations = append(recommendations, recommendation)
		}
	}
	return recommendations
}

func getLimitTargetRecommendations(vpaResource *vpa.VerticalPodAutoscaler, scfg *config.StrategyConfig) []TargetRecommendation {
	recommendations := []TargetRecommendation{}
	if vpaResource.Status.Recommendation != nil {
		for _, containerRecommendation := range vpaResource.Status.Recommendation.ContainerRecommendations {
			containerName := containerRecommendation.ContainerName
			recommendation := TargetRecommendation{
				ContainerName: containerName,
			}
			switch scfg.GetLimitCpuApplyTarget(containerName) {
			case config.LimitApplyTargetFrugal:
				recommendation.Cpu = containerRecommendation.LowerBound.Cpu()
			case config.LimitApplyTargetBalanced:
				recommendation.Cpu = containerRecommendation.Target.Cpu()
			case config.LimitApplyTargetPeak:
				recommendation.Cpu = containerRecommendation.UpperBound.Cpu()
			}
			switch scfg.GetLimitMemoryApplyTarget(containerName) {
			case config.LimitApplyTargetFrugal:
				recommendation.Memory = containerRecommendation.LowerBound.Memory()
			case config.LimitApplyTargetBalanced:
				recommendation.Memory = containerRecommendation.Target.Memory()
			case config.LimitApplyTargetPeak:
				recommendation.Memory = containerRecommendation.UpperBound.Memory()
			}
			recommendations = append(recommendations, recommendation)
		}
	}
	return recommendations
}
