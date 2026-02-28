// Package mailer provides email sending via SMTP.
package mailer

import (
	"fmt"
	"net/smtp"
	"strings"

	"github.com/rocjay1/balance-tracker-web/backend/internal/config"
)

// Mailer sends emails using SMTP credentials.
type Mailer struct {
	cfg config.SMTPConfig
	pwd string
}

// New creates a Mailer with the given SMTP configuration and password.
func New(cfg config.SMTPConfig, password string) *Mailer {
	return &Mailer{
		cfg: cfg,
		pwd: password,
	}
}

// Send delivers an email with the given subject and body to the specified recipients.
func (m *Mailer) Send(to []string, subject, body string) error {
	auth := smtp.PlainAuth("", m.cfg.User, m.pwd, m.cfg.Host)

	var b strings.Builder
	fmt.Fprintf(&b, "From: %s\r\n", m.cfg.User)
	fmt.Fprintf(&b, "To: %s\r\n", strings.Join(to, ","))
	fmt.Fprintf(&b, "Subject: %s\r\n", subject)
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/plain; charset=\"utf-8\"\r\n")
	b.WriteString("\r\n")
	b.WriteString(body)

	addr := fmt.Sprintf("%s:%d", m.cfg.Host, m.cfg.Port)
	if err := smtp.SendMail(addr, auth, m.cfg.User, to, []byte(b.String())); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}
