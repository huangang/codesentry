package services

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/huangang/codesentry/backend/internal/models"
	"gorm.io/gorm"
)

type NotificationService struct {
	db *gorm.DB
}

func NewNotificationService(db *gorm.DB) *NotificationService {
	return &NotificationService{db: db}
}

type ReviewNotification struct {
	ProjectName   string
	Branch        string
	Author        string
	CommitMessage string
	Score         float64
	ReviewResult  string
	EventType     string
	MRURL         string
}

func (s *NotificationService) SendReviewNotification(project *models.Project, notification *ReviewNotification) error {
	if !project.IMEnabled || project.IMBotID == nil {
		log.Printf("[Notification] IM notification disabled for project %d", project.ID)
		return nil
	}

	var bot models.IMBot
	if err := s.db.First(&bot, *project.IMBotID).Error; err != nil {
		return fmt.Errorf("IM bot not found: %w", err)
	}

	if !bot.IsActive {
		log.Printf("[Notification] IM bot %d is not active", bot.ID)
		return nil
	}

	log.Printf("[Notification] Sending notification to bot %s (type: %s)", bot.Name, bot.Type)

	var err error
	switch bot.Type {
	case "wechat_work":
		err = s.sendWeComNotification(&bot, notification)
	case "dingtalk":
		err = s.sendDingTalkNotification(&bot, notification)
	case "feishu":
		err = s.sendFeishuNotification(&bot, notification)
	case "slack":
		err = s.sendSlackNotification(&bot, notification)
	default:
		err = s.sendGenericWebhook(&bot, notification)
	}

	if err != nil {
		log.Printf("[Notification] Failed to send notification: %v", err)
		return err
	}

	log.Printf("[Notification] Notification sent successfully")
	return nil
}

func (s *NotificationService) buildMessage(n *ReviewNotification) string {
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

	reviewResult := n.ReviewResult

	msg := fmt.Sprintf(`📋 **Code Review Report**

**Project**: %s
**Event**: %s
**Branch**: %s
**Author**: %s
**Commit**: %s

%s **Score**: %.0f/100

---
%s`, n.ProjectName, eventTypeText, n.Branch, n.Author, commitMsg, scoreEmoji, n.Score, reviewResult)

	if n.MRURL != "" {
		msg += fmt.Sprintf("\n\n🔗 [View MR/PR](%s)", n.MRURL)
	}

	return msg
}

func (s *NotificationService) sendWeComNotification(bot *models.IMBot, n *ReviewNotification) error {
	msg := s.buildMessage(n)
	const maxLen = 4000

	if len(msg) <= maxLen {
		payload := map[string]interface{}{
			"msgtype": "markdown",
			"markdown": map[string]string{
				"content": msg,
			},
		}
		return s.postJSON(bot.Webhook, payload)
	}

	parts := s.splitMessage(msg, maxLen)
	for i, part := range parts {
		content := part
		if len(parts) > 1 {
			content = fmt.Sprintf("**[%d/%d]**\n\n%s", i+1, len(parts), part)
		}
		payload := map[string]interface{}{
			"msgtype": "markdown",
			"markdown": map[string]string{
				"content": content,
			},
		}
		if err := s.postJSON(bot.Webhook, payload); err != nil {
			return err
		}
	}
	return nil
}

func (s *NotificationService) sendDingTalkNotification(bot *models.IMBot, n *ReviewNotification) error {
	msg := s.buildMessage(n)
	const maxLen = 19000

	webhookURL := bot.Webhook
	if bot.Secret != "" {
		timestamp := time.Now().UnixMilli()
		sign := s.dingTalkSign(timestamp, bot.Secret)
		webhookURL = fmt.Sprintf("%s&timestamp=%d&sign=%s", bot.Webhook, timestamp, url.QueryEscape(sign))
	}

	if len(msg) <= maxLen {
		payload := map[string]interface{}{
			"msgtype": "markdown",
			"markdown": map[string]string{
				"title": fmt.Sprintf("Code Review: %s", n.ProjectName),
				"text":  msg,
			},
		}
		return s.postJSON(webhookURL, payload)
	}

	parts := s.splitMessage(msg, maxLen)
	for i, part := range parts {
		title := fmt.Sprintf("Code Review: %s [%d/%d]", n.ProjectName, i+1, len(parts))
		payload := map[string]interface{}{
			"msgtype": "markdown",
			"markdown": map[string]string{
				"title": title,
				"text":  part,
			},
		}
		if err := s.postJSON(webhookURL, payload); err != nil {
			return err
		}
	}
	return nil
}

func (s *NotificationService) dingTalkSign(timestamp int64, secret string) string {
	stringToSign := fmt.Sprintf("%d\n%s", timestamp, secret)
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(stringToSign))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

func (s *NotificationService) sendFeishuNotification(bot *models.IMBot, n *ReviewNotification) error {
	msg := s.buildMessage(n)
	const maxLen = 4000

	sendPart := func(content string) error {
		webhookURL := bot.Webhook
		if bot.Secret != "" {
			timestamp := time.Now().Unix()
			sign := s.feishuSign(timestamp, bot.Secret)
			payload := map[string]interface{}{
				"timestamp": fmt.Sprintf("%d", timestamp),
				"sign":      sign,
				"msg_type":  "text",
				"content": map[string]string{
					"text": content,
				},
			}
			return s.postJSON(webhookURL, payload)
		}
		payload := map[string]interface{}{
			"msg_type": "text",
			"content": map[string]string{
				"text": content,
			},
		}
		return s.postJSON(webhookURL, payload)
	}

	if len(msg) <= maxLen {
		return sendPart(msg)
	}

	parts := s.splitMessage(msg, maxLen)
	for i, part := range parts {
		content := part
		if len(parts) > 1 {
			content = fmt.Sprintf("[%d/%d]\n\n%s", i+1, len(parts), part)
		}
		if err := sendPart(content); err != nil {
			return err
		}
	}
	return nil
}

func (s *NotificationService) feishuSign(timestamp int64, secret string) string {
	stringToSign := fmt.Sprintf("%d\n%s", timestamp, secret)
	h := hmac.New(sha256.New, []byte(stringToSign))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

func (s *NotificationService) sendSlackNotification(bot *models.IMBot, n *ReviewNotification) error {
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
		return s.postJSON(bot.Webhook, payload)
	}

	parts := s.splitMessage(reviewResult, maxLen)
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
		if err := s.postJSON(bot.Webhook, payload); err != nil {
			return err
		}
	}
	return nil
}

func (s *NotificationService) sendGenericWebhook(bot *models.IMBot, n *ReviewNotification) error {
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
	return s.postJSON(bot.Webhook, payload)
}

// splitMessage splits a long message into chunks, trying to break at newlines
func (s *NotificationService) splitMessage(msg string, maxLen int) []string {
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

		// Try to find a good break point (newline) within the limit
		chunk := remaining[:maxLen]
		breakPoint := maxLen

		// Look for the last newline in the chunk
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

func (s *NotificationService) postJSON(webhookURL string, payload interface{}) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	log.Printf("[Notification] POST %s, payload length: %d", webhookURL, len(body))

	resp, err := http.Post(webhookURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	log.Printf("[Notification] Response: %d - %s", resp.StatusCode, string(respBody))

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}
