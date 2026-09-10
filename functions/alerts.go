package functions

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/sirupsen/logrus"
)

// AlertCandidate represents a single process considered for (or subject to) an OOM kill.
type AlertCandidate struct {
	Pid           int     `json:"pid"`
	User          string  `json:"user"`
	Command       string  `json:"command"`
	RSSBytes      int64   `json:"rss_bytes"`
	MemoryPercent float64 `json:"memory_percent"`
	Threshold     string  `json:"threshold"`
	Reason        string  `json:"reason"`
}

// AlertPayload is the JSON body posted to the /alerts/ endpoint.
type AlertPayload struct {
	Event       string           `json:"event"`
	Customer    string           `json:"customer"`
	ServerID    string           `json:"serverid"`
	SDPInstance string           `json:"sdp_instance"`
	Timestamp   string           `json:"timestamp"`
	Candidates  []AlertCandidate `json:"candidates"`
}

// HandleAlerts processes incoming OOM kill candidate/kill notifications and forwards them to Slack.
func HandleAlerts(w http.ResponseWriter, req *http.Request, logger *logrus.Logger, config *Config) {
	if req.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user, pass, ok := req.BasicAuth()
	if !ok || !VerifyUserPass(user, pass) {
		w.Header().Set("WWW-Authenticate", `Basic realm="api"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var payload AlertPayload
	decoder := json.NewDecoder(req.Body)
	if err := decoder.Decode(&payload); err != nil {
		logger.Errorf("Error decoding alert payload: %v", err)
		http.Error(w, "Failed to decode JSON data", http.StatusBadRequest)
		return
	}

	logger.Infof("Received alert %q for customer=%s serverid=%s sdp_instance=%s: %d kill candidate(s) identified",
		payload.Event, payload.Customer, payload.ServerID, payload.SDPInstance, len(payload.Candidates))
	for _, c := range payload.Candidates {
		logger.Infof("Memlimit kill candidate: PID %d user=%s cmd=%s reason=%s threshold=%s usage=%s (%.1f%%)",
			c.Pid, c.User, c.Command, c.Reason, c.Threshold, humanizeBytes(c.RSSBytes), c.MemoryPercent)
	}

	if err := SendSlackAlert(config, logger, payload); err != nil {
		logger.Errorf("Error sending Slack alert: %v", err)
		http.Error(w, "Failed to send Slack notification", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Alert processed\n")
}

// SendSlackAlert formats the alert payload and posts it to Slack, if enabled in config.
func SendSlackAlert(config *Config, logger *logrus.Logger, payload AlertPayload) error {
	slackCfg := config.Notifications.Slack
	if !slackCfg.Enabled {
		logger.Debug("Slack notifications disabled, skipping")
		return nil
	}

	message := formatSlackMessage(payload)

	if strings.EqualFold(slackCfg.Mode, "bot") {
		return sendSlackBotMessage(slackCfg, message)
	}
	return sendSlackWebhookMessage(slackCfg, message)
}

func formatSlackMessage(payload AlertPayload) string {
	var b strings.Builder
	fmt.Fprintf(&b, "*%s* customer=%s serverid=%s sdp_instance=%s at %s\n",
		payload.Event, payload.Customer, payload.ServerID, payload.SDPInstance, payload.Timestamp)
	for _, c := range payload.Candidates {
		fmt.Fprintf(&b, "> PID %d user=%s cmd=%s reason=%s threshold=%s usage=%s (%.1f%%)\n",
			c.Pid, c.User, c.Command, c.Reason, c.Threshold, humanizeBytes(c.RSSBytes), c.MemoryPercent)
	}
	return b.String()
}

func sendSlackWebhookMessage(slackCfg SlackConfig, message string) error {
	if slackCfg.WebhookURL == "" {
		return fmt.Errorf("slack webhook_url not configured")
	}
	body, err := json.Marshal(map[string]string{"text": message})
	if err != nil {
		return err
	}
	resp, err := http.Post(slackCfg.WebhookURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("error posting to Slack webhook: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("slack webhook returned status %d", resp.StatusCode)
	}
	return nil
}

func sendSlackBotMessage(slackCfg SlackConfig, message string) error {
	if slackCfg.BotToken == "" || slackCfg.ChannelID == "" {
		return fmt.Errorf("slack bot_token and channel_id must be configured for bot mode")
	}
	body, err := json.Marshal(map[string]string{
		"channel": slackCfg.ChannelID,
		"text":    message,
	})
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, "https://slack.com/api/chat.postMessage", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Authorization", "Bearer "+slackCfg.BotToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("error posting to Slack chat.postMessage: %v", err)
	}
	defer resp.Body.Close()

	var result struct {
		OK    bool   `json:"ok"`
		Error string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("error decoding Slack response: %v", err)
	}
	if !result.OK {
		return fmt.Errorf("slack API error: %s", result.Error)
	}
	return nil
}

// humanizeBytes converts a byte count into a human-readable binary-unit string (e.g. 2.0GiB).
func humanizeBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%dB", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%ciB", float64(b)/float64(div), "KMGTPE"[exp])
}
