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

func ReportUpdated(update *UpdateResult, scfg *config.StrategyConfig) {
	if update == nil || len(update.Changes) == 0 {
		return
	}
	klog.Infof("Updated: %s", scfg.Key)
	for _, update := range update.Changes {
		typeLabel := GetUpdateTypeLabel(update.Type)
		oldValueText := getResourceValueText(update.Type, update.Old)
		newValueText := getResourceValueText(update.Type, update.New)
		klog.Infof("Setting %s to %s (previously %s) for %s container: %s", typeLabel, newValueText, oldValueText, scfg.Key, update.ContainerName)
	}
	sendUpdatesToMattermost(update)
}
