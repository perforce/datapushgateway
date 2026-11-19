package functions

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfigExpandsSlackCredentials(t *testing.T) {
	oldPath, oldBin := p4ConfigPath, p4Bin
	t.Cleanup(func() {
		p4ConfigPath, p4Bin = oldPath, oldBin
	})

	const webhookURL = "https://example.invalid/webhook/test"
	const botToken = "test-token"
	t.Setenv("DATAPUSHGATEWAY_SLACK_WEBHOOK_URL", webhookURL)
	t.Setenv("DATAPUSHGATEWAY_SLACK_BOT_TOKEN", botToken)

	configPath := filepath.Join("..", "datapushgateway_config.yaml")
	config, err := LoadConfig(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if slack := config.Notifications.Slack; slack.WebhookURL != webhookURL || slack.BotToken != botToken {
		t.Fatal("Slack credentials did not resolve from the environment")
	}

	t.Setenv("DATAPUSHGATEWAY_SLACK_WEBHOOK_URL", "")
	t.Setenv("DATAPUSHGATEWAY_SLACK_BOT_TOKEN", "")
	config, err = LoadConfig(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if slack := config.Notifications.Slack; slack.WebhookURL != "" || slack.BotToken != "" {
		t.Fatal("empty environment values should leave Slack credentials unconfigured")
	}
}

func TestSlackWebhookErrorDoesNotExposeURL(t *testing.T) {
	const secret = "test-webhook-secret"
	err := sendSlackWebhookMessage(SlackConfig{WebhookURL: "invalid://example.invalid/" + secret}, "alert")
	if err == nil {
		t.Fatal("expected an error for an unsupported URL scheme")
	}
	if strings.Contains(err.Error(), secret) {
		t.Fatal("webhook error exposed the secret URL")
	}
}
