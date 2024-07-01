package target

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

func getTargetRecommandations(vpaResource *vpa.VerticalPodAutoscaler, vcfg *config.VpaWorkloadCfg) []TargetRecommandation {
	recommandations := []TargetRecommandation{}
	if vpaResource.Status.Recommendation != nil {
		for _, containerRecommendation := range vpaResource.Status.Recommendation.ContainerRecommendations {
			containerName := containerRecommendation.ContainerName
			recommandation := TargetRecommandation{
				ContainerName: containerName,
			}
			switch vcfg.GetRequestCpuApplyTarget(containerName) {
			case config.ApplyTargetFrugal:
				recommandation.Cpu = containerRecommendation.LowerBound.Cpu()
			case config.ApplyTargetBalanced:
				recommandation.Cpu = containerRecommendation.Target.Cpu()
			case config.ApplyTargetPeak:
				recommandation.Cpu = containerRecommendation.UpperBound.Cpu()
			}
			switch vcfg.GetRequestMemoryApplyTarget(containerName) {
			case config.ApplyTargetFrugal:
				recommandation.Memory = containerRecommendation.LowerBound.Memory()
			case config.ApplyTargetBalanced:
				recommandation.Memory = containerRecommendation.Target.Memory()
			case config.ApplyTargetPeak:
				recommandation.Memory = containerRecommendation.UpperBound.Memory()
			}
			recommandations = append(recommandations, recommandation)
		}
	}
	return recommandations
}
