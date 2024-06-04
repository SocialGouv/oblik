package controller

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type Payload struct {
	Text string `json:"text"`
}

func sendMattermostAlert(message string) error {
	webhookURL := getEnv("OBLIK_MATTERMOST_WEBHOOK_URL", "")

	payload := fmt.Sprintf(`{"text": "%s"}`, message)

	formData := url.Values{}
	formData.Set("payload", payload)

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
