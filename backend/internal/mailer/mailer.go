package mailer

import (
	"fmt"
	"net/smtp"
	"strings"

	"github.com/roccodavino/balance-tracker-web/backend/internal/config"
)

type Mailer struct {
	cfg config.SMTPConfig
	pwd string
}

func New(cfg config.SMTPConfig, password string) *Mailer {
	return &Mailer{
		cfg: cfg,
		pwd: password,
	}
}

func (m *Mailer) Send(to []string, subject, body string) error {
	auth := smtp.PlainAuth("", m.cfg.User, m.pwd, m.cfg.Host)
	
	msg := fmt.Sprintf("From: %s\r\n", m.cfg.User)
	msg += fmt.Sprintf("To: %s\r\n", strings.Join(to, ","))
	msg += fmt.Sprintf("Subject: %s\r\n", subject)
	msg += "MIME-Version: 1.0\r\n"
	msg += "Content-Type: text/plain; charset=\"utf-8\"\r\n"
	msg += "\r\n" + body

	addr := fmt.Sprintf("%s:%d", m.cfg.Host, m.cfg.Port)
	if err := smtp.SendMail(addr, auth, m.cfg.User, to, []byte(msg)); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}
