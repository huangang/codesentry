package services

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/huangang/codesentry/backend/internal/models"
	"github.com/huangang/codesentry/backend/pkg/logger"
)

// NotificationAdapter defines the interface for sending notifications to different IM platforms.
// Each adapter handles the specific payload format and signing requirements of its platform.
type NotificationAdapter interface {
	SendRichMessage(webhook string, bot *models.IMBot, notification *ReviewNotification) error
	SendTextMessage(webhook string, bot *models.IMBot, message string) error
}

// getAdapter returns the appropriate notification adapter for the given bot type
func getAdapter(botType string) NotificationAdapter {
	switch botType {
	case "wechat_work":
		return &wecomAdapter{}
	case "dingtalk":
		return &dingtalkAdapter{}
	case "feishu":
		return &feishuAdapter{}
	case "slack":
		return &slackAdapter{}
	case "discord":
		return &discordAdapter{}
	case "teams":
		return &teamsAdapter{}
	case "telegram":
		return &telegramAdapter{}
	default:
		return &genericAdapter{}
	}
}

// --- Helper functions shared by adapters ---

func postJSONWithClient(client *http.Client, webhookURL string, payload interface{}) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	logger.Infof("[Notification] POST %s, payload length: %d", webhookURL, len(body))

	req, err := http.NewRequest("POST", webhookURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	logger.Infof("[Notification] Response: %d - %s", resp.StatusCode, string(respBody))

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

var notificationHTTPClient = &http.Client{Timeout: 10 * time.Second}

func splitMessage(msg string, maxLen int) []string {
	if len(msg) <= maxLen {
		return []string{msg}
	}

	var parts []string
	remaining := msg

	for len(remaining) > 0 {
		if len(remaining) <= maxLen {
			parts = append(parts, remaining)
			break
		}

		chunk := remaining[:maxLen]
		breakPoint := maxLen

		for i := len(chunk) - 1; i > maxLen/2; i-- {
			if chunk[i] == '\n' {
				breakPoint = i + 1
				break
			}
		}

		parts = append(parts, remaining[:breakPoint])
		remaining = remaining[breakPoint:]
	}

	return parts
}

func buildMessage(n *ReviewNotification) string {
	scoreEmoji := "🟢"
	if n.Score < 60 {
		scoreEmoji = "🔴"
	} else if n.Score < 80 {
		scoreEmoji = "🟡"
	}

	eventTypeText := "Push"
	if n.EventType == "merge_request" {
		eventTypeText = "Merge Request"
	}

	commitMsg := n.CommitMessage
	if len(commitMsg) > 100 {
		commitMsg = commitMsg[:100] + "..."
	}

	msg := fmt.Sprintf(`📋 **Code Review Report**

**Project**: %s
**Event**: %s
**Branch**: %s
**Author**: %s
**Commit**: %s

%s **Score**: %.0f/100

---
%s`, n.ProjectName, eventTypeText, n.Branch, n.Author, commitMsg, scoreEmoji, n.Score, n.ReviewResult)

	if n.MRURL != "" {
		msg += fmt.Sprintf("\n\n🔗 [View MR/PR](%s)", n.MRURL)
	}

	return msg
}

func dingTalkSign(timestamp int64, secret string) string {
	stringToSign := fmt.Sprintf("%d\n%s", timestamp, secret)
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(stringToSign))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

func feishuSign(timestamp int64, secret string) string {
	stringToSign := fmt.Sprintf("%d\n%s", timestamp, secret)
	h := hmac.New(sha256.New, []byte(stringToSign))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

func dingTalkWebhookURL(webhook, secret string) string {
	if secret == "" {
		return webhook
	}
	timestamp := time.Now().UnixMilli()
	sign := dingTalkSign(timestamp, secret)
	return fmt.Sprintf("%s&timestamp=%d&sign=%s", webhook, timestamp, url.QueryEscape(sign))
}

// --- Adapter implementations ---

// wecomAdapter handles WeCom (Enterprise WeChat) bot notifications
type wecomAdapter struct{}

func (a *wecomAdapter) SendRichMessage(webhook string, bot *models.IMBot, n *ReviewNotification) error {
	msg := buildMessage(n)
	const maxLen = 4000

	if len(msg) <= maxLen {
		payload := map[string]interface{}{
			"msgtype": "markdown_v2",
			"markdown_v2": map[string]string{
				"content": msg,
			},
		}
		return postJSONWithClient(notificationHTTPClient, webhook, payload)
	}

	parts := splitMessage(msg, maxLen)
	for i, part := range parts {
		content := part
		if len(parts) > 1 {
			content = fmt.Sprintf("**[%d/%d]**\n\n%s", i+1, len(parts), part)
		}
		payload := map[string]interface{}{
			"msgtype": "markdown_v2",
			"markdown_v2": map[string]string{
				"content": content,
			},
		}
		if err := postJSONWithClient(notificationHTTPClient, webhook, payload); err != nil {
			return err
		}
	}
	return nil
}

func (a *wecomAdapter) SendTextMessage(webhook string, bot *models.IMBot, message string) error {
	payload := map[string]interface{}{
		"msgtype": "markdown_v2",
		"markdown_v2": map[string]string{
			"content": message,
		},
	}
	return postJSONWithClient(notificationHTTPClient, webhook, payload)
}

// dingtalkAdapter handles DingTalk bot notifications
type dingtalkAdapter struct{}

func (a *dingtalkAdapter) SendRichMessage(webhook string, bot *models.IMBot, n *ReviewNotification) error {
	msg := buildMessage(n)
	const maxLen = 19000

	webhookURL := dingTalkWebhookURL(bot.Webhook, bot.Secret)

	if len(msg) <= maxLen {
		payload := map[string]interface{}{
			"msgtype": "markdown",
			"markdown": map[string]string{
				"title": fmt.Sprintf("Code Review: %s", n.ProjectName),
				"text":  msg,
			},
		}
		return postJSONWithClient(notificationHTTPClient, webhookURL, payload)
	}

	parts := splitMessage(msg, maxLen)
	for i, part := range parts {
		title := fmt.Sprintf("Code Review: %s [%d/%d]", n.ProjectName, i+1, len(parts))
		payload := map[string]interface{}{
			"msgtype": "markdown",
			"markdown": map[string]string{
				"title": title,
				"text":  part,
			},
		}
		if err := postJSONWithClient(notificationHTTPClient, webhookURL, payload); err != nil {
			return err
		}
	}
	return nil
}

func (a *dingtalkAdapter) SendTextMessage(webhook string, bot *models.IMBot, message string) error {
	webhookURL := dingTalkWebhookURL(bot.Webhook, bot.Secret)
	payload := map[string]interface{}{
		"msgtype": "markdown",
		"markdown": map[string]string{
			"title": "System Error Alert",
			"text":  message,
		},
	}
	return postJSONWithClient(notificationHTTPClient, webhookURL, payload)
}

// feishuAdapter handles Feishu (Lark) bot notifications
type feishuAdapter struct{}

func (a *feishuAdapter) sendFeishu(webhook, secret, content string) error {
	if secret != "" {
		timestamp := time.Now().Unix()
		sign := feishuSign(timestamp, secret)
		payload := map[string]interface{}{
			"timestamp": fmt.Sprintf("%d", timestamp),
			"sign":      sign,
			"msg_type":  "text",
			"content": map[string]string{
				"text": content,
			},
		}
		return postJSONWithClient(notificationHTTPClient, webhook, payload)
	}
	payload := map[string]interface{}{
		"msg_type": "text",
		"content": map[string]string{
			"text": content,
		},
	}
	return postJSONWithClient(notificationHTTPClient, webhook, payload)
}

func (a *feishuAdapter) SendRichMessage(webhook string, bot *models.IMBot, n *ReviewNotification) error {
	msg := buildMessage(n)
	const maxLen = 4000

	if len(msg) <= maxLen {
		return a.sendFeishu(webhook, bot.Secret, msg)
	}

	parts := splitMessage(msg, maxLen)
	for i, part := range parts {
		content := part
		if len(parts) > 1 {
			content = fmt.Sprintf("[%d/%d]\n\n%s", i+1, len(parts), part)
		}
		if err := a.sendFeishu(webhook, bot.Secret, content); err != nil {
			return err
		}
	}
	return nil
}

func (a *feishuAdapter) SendTextMessage(webhook string, bot *models.IMBot, message string) error {
	return a.sendFeishu(webhook, bot.Secret, message)
}

// slackAdapter handles Slack bot notifications
type slackAdapter struct{}

func (a *slackAdapter) SendRichMessage(webhook string, bot *models.IMBot, n *ReviewNotification) error {
	scoreEmoji := ":large_green_circle:"
	if n.Score < 60 {
		scoreEmoji = ":red_circle:"
	} else if n.Score < 80 {
		scoreEmoji = ":large_yellow_circle:"
	}

	header := fmt.Sprintf("*Code Review Report*\n*Project*: %s\n*Branch*: %s\n*Author*: %s\n%s *Score*: %.0f/100",
		n.ProjectName, n.Branch, n.Author, scoreEmoji, n.Score)

	const maxLen = 3000
	reviewResult := n.ReviewResult

	if len(reviewResult) <= maxLen {
		payload := map[string]interface{}{
			"text": header,
			"blocks": []map[string]interface{}{
				{
					"type": "section",
					"text": map[string]string{
						"type": "mrkdwn",
						"text": header,
					},
				},
				{
					"type": "section",
					"text": map[string]string{
						"type": "mrkdwn",
						"text": reviewResult,
					},
				},
			},
		}
		return postJSONWithClient(notificationHTTPClient, webhook, payload)
	}

	parts := splitMessage(reviewResult, maxLen)
	for i, part := range parts {
		title := header
		if i > 0 {
			title = fmt.Sprintf("*Code Review Report [%d/%d]*", i+1, len(parts))
		} else if len(parts) > 1 {
			title = fmt.Sprintf("%s\n_(%d parts total)_", header, len(parts))
		}
		payload := map[string]interface{}{
			"text": title,
			"blocks": []map[string]interface{}{
				{
					"type": "section",
					"text": map[string]string{
						"type": "mrkdwn",
						"text": title,
					},
				},
				{
					"type": "section",
					"text": map[string]string{
						"type": "mrkdwn",
						"text": part,
					},
				},
			},
		}
		if err := postJSONWithClient(notificationHTTPClient, webhook, payload); err != nil {
			return err
		}
	}
	return nil
}

func (a *slackAdapter) SendTextMessage(webhook string, bot *models.IMBot, message string) error {
	payload := map[string]interface{}{
		"text": message,
	}
	return postJSONWithClient(notificationHTTPClient, webhook, payload)
}

// discordAdapter handles Discord webhook notifications
type discordAdapter struct{}

func (a *discordAdapter) SendRichMessage(webhook string, bot *models.IMBot, n *ReviewNotification) error {
	msg := buildMessage(n)
	payload := map[string]interface{}{
		"content": msg,
	}
	return postJSONWithClient(notificationHTTPClient, webhook, payload)
}

func (a *discordAdapter) SendTextMessage(webhook string, bot *models.IMBot, message string) error {
	payload := map[string]interface{}{
		"content": message,
	}
	return postJSONWithClient(notificationHTTPClient, webhook, payload)
}

// teamsAdapter handles Microsoft Teams webhook notifications
type teamsAdapter struct{}

func buildAdaptiveCard(text string) map[string]interface{} {
	return map[string]interface{}{
		"type": "message",
		"attachments": []map[string]interface{}{
			{
				"contentType": "application/vnd.microsoft.card.adaptive",
				"content": map[string]interface{}{
					"type":    "AdaptiveCard",
					"$schema": "http://adaptivecards.io/schemas/adaptive-card.json",
					"version": "1.5",
					"body": []map[string]interface{}{
						{
							"type": "TextBlock",
							"text": text,
							"wrap": true,
						},
					},
				},
			},
		},
	}
}

func (a *teamsAdapter) SendRichMessage(webhook string, bot *models.IMBot, n *ReviewNotification) error {
	msg := buildMessage(n)
	return postJSONWithClient(notificationHTTPClient, webhook, buildAdaptiveCard(msg))
}

func (a *teamsAdapter) SendTextMessage(webhook string, bot *models.IMBot, message string) error {
	return postJSONWithClient(notificationHTTPClient, webhook, buildAdaptiveCard(message))
}

// telegramAdapter handles Telegram bot notifications
type telegramAdapter struct{}

func (a *telegramAdapter) sendTelegram(webhook, chatID, text string) error {
	if chatID == "" {
		return fmt.Errorf("telegram chat_id is required in extra field")
	}
	payload := map[string]interface{}{
		"chat_id":    chatID,
		"text":       text,
		"parse_mode": "Markdown",
	}
	return postJSONWithClient(notificationHTTPClient, webhook, payload)
}

func (a *telegramAdapter) SendRichMessage(webhook string, bot *models.IMBot, n *ReviewNotification) error {
	msg := buildMessage(n)
	return a.sendTelegram(webhook, bot.Extra, msg)
}

func (a *telegramAdapter) SendTextMessage(webhook string, bot *models.IMBot, message string) error {
	return a.sendTelegram(webhook, bot.Extra, message)
}

// genericAdapter handles generic webhook notifications
type genericAdapter struct{}

func (a *genericAdapter) SendRichMessage(webhook string, bot *models.IMBot, n *ReviewNotification) error {
	payload := map[string]interface{}{
		"project":        n.ProjectName,
		"branch":         n.Branch,
		"author":         n.Author,
		"commit_message": n.CommitMessage,
		"score":          n.Score,
		"review_result":  n.ReviewResult,
		"event_type":     n.EventType,
		"mr_url":         n.MRURL,
	}
	return postJSONWithClient(notificationHTTPClient, webhook, payload)
}

func (a *genericAdapter) SendTextMessage(webhook string, bot *models.IMBot, message string) error {
	payload := map[string]interface{}{
		"type":    "error",
		"message": message,
	}
	return postJSONWithClient(notificationHTTPClient, webhook, payload)
}
