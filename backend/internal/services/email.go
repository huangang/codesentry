package services

import (
	"crypto/tls"
	"fmt"
	"github.com/huangang/codesentry/backend/pkg/logger"
	"net/smtp"
	"strconv"
	"strings"

	"github.com/huangang/codesentry/backend/internal/models"
	"gorm.io/gorm"
)

type EmailService struct {
	db *gorm.DB
}

type EmailConfig struct {
	Enabled  bool
	Host     string
	Port     int
	Username string
	Password string
	From     string
	UseTLS   bool
}

func NewEmailService(db *gorm.DB) *EmailService {
	return &EmailService{db: db}
}

func (s *EmailService) GetConfig() *EmailConfig {
	config := &EmailConfig{}

	var configs []models.SystemConfig
	s.db.Where("`group` = ?", "email").Find(&configs)

	for _, c := range configs {
		switch c.Key {
		case "email_enabled":
			config.Enabled = c.Value == "true"
		case "email_host":
			config.Host = c.Value
		case "email_port":
			if port, err := strconv.Atoi(c.Value); err == nil {
				config.Port = port
			}
		case "email_username":
			config.Username = c.Value
		case "email_password":
			config.Password = c.Value
		case "email_from":
			config.From = c.Value
		case "email_use_tls":
			config.UseTLS = c.Value == "true"
		}
	}

	if config.Port == 0 {
		config.Port = 587
	}

	return config
}

func (s *EmailService) SendReviewNotification(notification *ReviewNotification, recipients []string) error {
	config := s.GetConfig()
	if !config.Enabled || config.Host == "" {
		return nil
	}

	if len(recipients) == 0 {
		return nil
	}

	subject := fmt.Sprintf("[CodeSentry] Code Review: %s - Score %.0f", notification.ProjectName, notification.Score)

	body := s.buildEmailBody(notification)

	return s.sendEmail(config, recipients, subject, body)
}

func (s *EmailService) buildEmailBody(n *ReviewNotification) string {
	var sb strings.Builder

	sb.WriteString("<html><body style=\"font-family: Arial, sans-serif;\">")
	sb.WriteString("<h2>🤖 AI Code Review Result</h2>")
	sb.WriteString("<table style=\"border-collapse: collapse; margin-bottom: 20px;\">")

	rows := []struct{ label, value string }{
		{"Project", n.ProjectName},
		{"Branch", n.Branch},
		{"Author", n.Author},
		{"Event", n.EventType},
		{"Score", fmt.Sprintf("%.0f / 100", n.Score)},
	}

	for _, r := range rows {
		sb.WriteString(fmt.Sprintf("<tr><td style=\"padding: 8px; border: 1px solid #ddd; font-weight: bold;\">%s</td><td style=\"padding: 8px; border: 1px solid #ddd;\">%s</td></tr>", r.label, r.value))
	}
	sb.WriteString("</table>")

	sb.WriteString("<h3>Commit Message</h3>")
	sb.WriteString(fmt.Sprintf("<pre style=\"background: #f5f5f5; padding: 12px; border-radius: 4px;\">%s</pre>", n.CommitMessage))

	sb.WriteString("<h3>Review Result</h3>")
	sb.WriteString(fmt.Sprintf("<div style=\"background: #f9f9f9; padding: 16px; border-radius: 4px; white-space: pre-wrap;\">%s</div>", n.ReviewResult))

	if n.MRURL != "" {
		sb.WriteString(fmt.Sprintf("<p><a href=\"%s\">View MR/PR</a></p>", n.MRURL))
	}

	sb.WriteString("<hr><p style=\"color: #888; font-size: 12px;\">Powered by CodeSentry</p>")
	sb.WriteString("</body></html>")

	return sb.String()
}

func (s *EmailService) sendEmail(config *EmailConfig, to []string, subject, body string) error {
	from := config.From
	if from == "" {
		from = config.Username
	}

	headers := make(map[string]string)
	headers["From"] = from
	headers["To"] = strings.Join(to, ",")
	headers["Subject"] = subject
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "text/html; charset=UTF-8"

	var message strings.Builder
	for k, v := range headers {
		message.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}
	message.WriteString("\r\n")
	message.WriteString(body)

	addr := fmt.Sprintf("%s:%d", config.Host, config.Port)

	var auth smtp.Auth
	if config.Username != "" && config.Password != "" {
		auth = smtp.PlainAuth("", config.Username, config.Password, config.Host)
	}

	var err error
	if config.UseTLS {
		err = s.sendEmailTLS(config, addr, auth, from, to, message.String())
	} else {
		err = smtp.SendMail(addr, auth, from, to, []byte(message.String()))
	}

	if err != nil {
		logger.Infof("[Email] Failed to send email: %v", err)
		return err
	}

	logger.Infof("[Email] Sent notification to %v", to)
	return nil
}

func (s *EmailService) sendEmailTLS(config *EmailConfig, addr string, auth smtp.Auth, from string, to []string, message string) error {
	tlsConfig := &tls.Config{
		ServerName: config.Host,
	}

	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return err
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, config.Host)
	if err != nil {
		return err
	}
	defer client.Close()

	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return err
		}
	}

	if err := client.Mail(from); err != nil {
		return err
	}

	for _, addr := range to {
		if err := client.Rcpt(addr); err != nil {
			return err
		}
	}

	w, err := client.Data()
	if err != nil {
		return err
	}

	_, err = w.Write([]byte(message))
	if err != nil {
		return err
	}

	return w.Close()
}
