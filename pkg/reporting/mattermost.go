package reporting

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/SocialGouv/oblik/pkg/utils"
	"k8s.io/klog/v2"
)

type Payload struct {
	Text string `json:"text"`
}

func sendUpdatesToMattermost(update *UpdateResult) {
	if len(update.Changes) == 0 {
		return
	}

	markdown := []string{}

	var title string

	switch update.Type {
	case ResultTypeDryRun:
		title = fmt.Sprintf("👻 Dry Run - Changes on %s", update.Key)
	case ResultTypeFailed:
		title = fmt.Sprintf("⚠️ Failure on %s", update.Key)
	case ResultTypeSuccess:
		title = fmt.Sprintf("▶️ Changes on %s", update.Key)
	}

	markdown = append(
		markdown,
		title,
		"\n| Container Name | Change Type | Old Value | New Value |",
		"|:-----|------|------|------|",
	)

	for _, update := range update.Changes {
		typeLabel := GetUpdateTypeLabel(update.Type)
		oldValueText := getResourceValueText(update.Type, update.Old)
		newValueText := getResourceValueText(update.Type, update.New)
		markdown = append(markdown, "|"+update.ContainerName+"|"+typeLabel+"|"+oldValueText+"|"+newValueText+"|")
	}

	if update.Type == ResultTypeFailed && update.Error != nil {
		markdown = append(markdown, "---", fmt.Sprintf("Error: %s", update.Error.Error()))
	}

	if err := sendMattermostAlert(strings.Join(markdown, "\n")); err != nil {
		klog.Errorf("Error sending Mattermost alert: %s", err.Error())
	}
}

func sendMattermostAlert(message string) error {
	webhookURL := utils.GetEnv("OBLIK_MATTERMOST_WEBHOOK_URL", "")

	payload := Payload{Text: message}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("Error marshaling payload: %s\n", err)
	}

	payloadString := string(payloadJSON)

	formData := url.Values{}
	formData.Set("payload", payloadString)

	req, err := http.NewRequest("POST", webhookURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "Oblik")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("non-OK HTTP status: %v", resp.StatusCode)
	}

	return nil
}
