package controller

import (
	"fmt"
	"net/http"

	"github.com/go-resty/resty/v2"
)

type MattermostMessage struct {
	Text string `json:"text"`
}

func sendMattermostAlert(message string) error {
	client := resty.New()
	msg := MattermostMessage{Text: message}
	webhookURL := getEnv("OBLIK_MATTERMOST_WEBHOOK_URL", "")

	if webhookURL == "" {
		return nil
	}

	resp, err := client.R().
		SetHeader("Content-Type", "application/json").
		SetBody(msg).
		Post(webhookURL)

	if err != nil {
		return fmt.Errorf("error sending request: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("received non-OK response code: %d", resp.StatusCode())
	}

	return nil
}

// func main() {
// 	webhookURL := "https://your-mattermost-instance/hooks/your-webhook-id"
// 	message := "Alerte : quelque chose s'est passé !"

// 	if err := sendMattermostAlert(webhookURL, message); err != nil {
// 		fmt.Printf("Error sending alert: %v\n", err)
// 	} else {
// 		fmt.Println("Alerte envoyée avec succès !")
// 	}
// }
