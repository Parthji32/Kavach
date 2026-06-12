package alerts

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/parthjindal/kavach/internal/fingerprint"
	"github.com/parthjindal/kavach/internal/models"
)

// AlertService dispatches alerts through multiple channels
type AlertService struct {
	emailSender *EmailSender
	slackSender *SlackSender
}

// NewAlertService creates a new alert service
func NewAlertService() *AlertService {
	return &AlertService{
		emailSender: NewEmailSender(),
		slackSender: NewSlackSender(),
	}
}

// AlertPayload contains all info needed to send an alert
type AlertPayload struct {
	Token       *models.Token
	Fingerprint *fingerprint.CapturedFingerprint
	TriggeredAt time.Time
}

// Dispatch sends an alert through all configured channels
func (s *AlertService) Dispatch(payload AlertPayload) {
	log.Printf("🚨 Dispatching alert for token: %s (type: %s)", payload.Token.Name, payload.Token.Type)

	// Send email alert
	if s.emailSender.IsConfigured() {
		go func() {
			if err := s.emailSender.Send(payload); err != nil {
				log.Printf("❌ Email alert failed: %v", err)
			} else {
				log.Printf("✅ Email alert sent for token: %s", payload.Token.Name)
			}
		}()
	}

	// Send Slack alert
	if s.slackSender.IsConfigured() {
		go func() {
			if err := s.slackSender.Send(payload); err != nil {
				log.Printf("❌ Slack alert failed: %v", err)
			} else {
				log.Printf("✅ Slack alert sent for token: %s", payload.Token.Name)
			}
		}()
	}
}

// ========== EMAIL SENDER (Resend) ==========

// EmailSender sends email alerts via Resend API
type EmailSender struct {
	apiKey    string
	fromEmail string
}

// NewEmailSender creates a new email sender
func NewEmailSender() *EmailSender {
	return &EmailSender{
		apiKey:    os.Getenv("RESEND_API_KEY"),
		fromEmail: os.Getenv("ALERT_FROM_EMAIL"),
	}
}

// IsConfigured returns true if email alerts are configured
func (e *EmailSender) IsConfigured() bool {
	return e.apiKey != "" && e.fromEmail != ""
}

// Send sends an email alert via Resend
func (e *EmailSender) Send(payload AlertPayload) error {
	// Build email content — all user-controlled fields MUST be HTML-escaped
	// to prevent XSS via crafted User-Agent or other attacker-controlled headers
	subject := fmt.Sprintf("🚨 Kavach Alert: Token '%s' triggered!", payload.Token.Name)

	htmlBody := fmt.Sprintf(`
		<div style="font-family: -apple-system, BlinkMacSystemFont, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px;">
			<div style="background: #0A0A14; border-radius: 12px; padding: 24px; color: #E2E8F0;">
				<h1 style="color: #7C3AED; margin: 0 0 4px;">🛡️ Kavach Alert</h1>
				<p style="color: #64748B; margin: 0 0 20px; font-size: 14px;">A canary token was just triggered</p>
				
				<div style="background: rgba(239,68,68,0.1); border: 1px solid rgba(239,68,68,0.3); border-radius: 8px; padding: 16px; margin-bottom: 16px;">
					<p style="margin: 0; font-size: 14px;"><strong>Token:</strong> %s</p>
					<p style="margin: 4px 0 0; font-size: 14px;"><strong>Type:</strong> %s</p>
					<p style="margin: 4px 0 0; font-size: 14px;"><strong>Time:</strong> %s</p>
				</div>

				<div style="background: rgba(255,255,255,0.05); border-radius: 8px; padding: 16px;">
					<p style="margin: 0 0 8px; font-size: 12px; color: #64748B; text-transform: uppercase; letter-spacing: 0.05em;">Attacker Information</p>
					<p style="margin: 4px 0; font-size: 14px;"><strong>IP:</strong> %s</p>
					<p style="margin: 4px 0; font-size: 14px;"><strong>Location:</strong> %s, %s</p>
					<p style="margin: 4px 0; font-size: 14px;"><strong>Browser:</strong> %s</p>
					<p style="margin: 4px 0; font-size: 14px;"><strong>OS:</strong> %s</p>
					<p style="margin: 4px 0; font-size: 14px;"><strong>User Agent:</strong> <code style="font-size: 12px;">%s</code></p>
				</div>

				<div style="margin-top: 20px; text-align: center;">
					<a href="https://app.kavach.dev/alerts" style="background: #7C3AED; color: #ffffff; padding: 10px 24px; border-radius: 6px; text-decoration: none; font-weight: 600; font-size: 14px;">View in Dashboard</a>
				</div>
			</div>
		</div>`,
		html.EscapeString(payload.Token.Name),
		html.EscapeString(string(payload.Token.Type)),
		payload.TriggeredAt.Format("Jan 2, 2006 3:04 PM MST"),
		html.EscapeString(payload.Fingerprint.IPAddress),
		html.EscapeString(payload.Fingerprint.City), html.EscapeString(payload.Fingerprint.Country),
		html.EscapeString(payload.Fingerprint.Browser),
		html.EscapeString(payload.Fingerprint.OS),
		html.EscapeString(payload.Fingerprint.UserAgent),
	)

	// Send via Resend API
	reqBody := map[string]interface{}{
		"from":    e.fromEmail,
		"to":     []string{os.Getenv("ALERT_TO_EMAIL")}, // TODO: Get from user settings
		"subject": subject,
		"html":    htmlBody,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal email body: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.resend.com/emails", bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+e.apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("resend API returned status %d", resp.StatusCode)
	}

	return nil
}

// ========== SLACK SENDER ==========

// SlackSender sends alerts to Slack via webhook
type SlackSender struct {
	webhookURL string
}

// NewSlackSender creates a new Slack sender
func NewSlackSender() *SlackSender {
	return &SlackSender{
		webhookURL: os.Getenv("SLACK_WEBHOOK_URL"),
	}
}

// IsConfigured returns true if Slack alerts are configured
func (s *SlackSender) IsConfigured() bool {
	// Validate webhook URL format to prevent SSRF
	return s.webhookURL != "" && strings.HasPrefix(s.webhookURL, "https://hooks.slack.com/")
}

// Send sends an alert to Slack
func (s *SlackSender) Send(payload AlertPayload) error {
	// Build Slack message with blocks
	message := map[string]interface{}{
		"blocks": []map[string]interface{}{
			{
				"type": "header",
				"text": map[string]string{
					"type": "plain_text",
					"text": fmt.Sprintf("🚨 Token Triggered: %s", payload.Token.Name),
				},
			},
			{
				"type": "section",
				"fields": []map[string]string{
					{"type": "mrkdwn", "text": fmt.Sprintf("*Token Type:*\n%s", payload.Token.Type)},
					{"type": "mrkdwn", "text": fmt.Sprintf("*Time:*\n%s", payload.TriggeredAt.Format("3:04 PM"))},
					{"type": "mrkdwn", "text": fmt.Sprintf("*Attacker IP:*\n`%s`", payload.Fingerprint.IPAddress)},
					{"type": "mrkdwn", "text": fmt.Sprintf("*Location:*\n%s, %s", payload.Fingerprint.City, payload.Fingerprint.Country)},
					{"type": "mrkdwn", "text": fmt.Sprintf("*Browser:*\n%s / %s", payload.Fingerprint.Browser, payload.Fingerprint.OS)},
					{"type": "mrkdwn", "text": fmt.Sprintf("*Fingerprint:*\n`%s`", payload.Fingerprint.UniqueHash)},
				},
			},
			{
				"type": "actions",
				"elements": []map[string]interface{}{
					{
						"type": "button",
						"text": map[string]string{
							"type": "plain_text",
							"text": "View in Dashboard",
						},
						"url":   "https://app.kavach.dev/alerts",
						"style": "primary",
					},
				},
			},
		},
	}

	jsonBody, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal slack message: %w", err)
	}

	req, err := http.NewRequest("POST", s.webhookURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send slack message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("slack webhook returned status %d", resp.StatusCode)
	}

	return nil
}
