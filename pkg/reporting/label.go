package reporting

import (
	"github.com/SocialGouv/oblik/pkg/config"
	"github.com/SocialGouv/oblik/pkg/utils"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/klog/v2"
)

func GetUpdateTypeLabel(updateType UpdateType) string {
	switch updateType {
	case UpdateTypeCpuRequest:
		return "CPU request"
	case UpdateTypeMemoryRequest:
		return "Memory request"
	case UpdateTypeCpuLimit:
		return "CPU limit"
	case UpdateTypeMemoryLimit:
		return "Memory limit"
	}
	return ""
}

func getResourceValueText(updateType UpdateType, value resource.Quantity) string {
	switch updateType {
	case UpdateTypeMemoryLimit:
		return utils.FormatMemory(value)
	case UpdateTypeMemoryRequest:
		return utils.FormatMemory(value)
	default:
		return value.String()
	}
}

func ReportUpdated(updates []Update, vcfg *config.VpaWorkloadCfg) {
	if len(updates) == 0 {
		return
	}
	klog.Infof("Updated: %s", vcfg.Key)
	for _, update := range updates {
		typeLabel := GetUpdateTypeLabel(update.Type)
		oldValueText := getResourceValueText(update.Type, update.Old)
		newValueText := getResourceValueText(update.Type, update.New)
		klog.Infof("Setting %s to %s (previously %s) for %s container: %s", typeLabel, newValueText, oldValueText, vcfg.Key, update.ContainerName)
	}
	sendUpdatesToMattermost(updates, vcfg)
}
