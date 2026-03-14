package mail

import (
	"fmt"
	"net/smtp"
	"strings"

	"github.com/derangedhermits/website/internal/config"
)

type Mailer struct {
	cfg *config.Config
}

func New(cfg *config.Config) *Mailer {
	return &Mailer{cfg: cfg}
}

func (m *Mailer) Enabled() bool {
	return m.cfg.SMTPHost != ""
}

func (m *Mailer) Send(to, subject, htmlBody string) error {
	if !m.Enabled() {
		return fmt.Errorf("SMTP not configured")
	}

	addr := fmt.Sprintf("%s:%d", m.cfg.SMTPHost, m.cfg.SMTPPort)
	auth := smtp.PlainAuth("", m.cfg.SMTPUser, m.cfg.SMTPPass, m.cfg.SMTPHost)

	headers := map[string]string{
		"From":         m.cfg.SMTPFrom,
		"To":           to,
		"Subject":      subject,
		"MIME-Version": "1.0",
		"Content-Type": "text/html; charset=\"UTF-8\"",
	}

	var msg strings.Builder
	for k, v := range headers {
		fmt.Fprintf(&msg, "%s: %s\r\n", k, v)
	}
	msg.WriteString("\r\n")
	msg.WriteString(htmlBody)

	return smtp.SendMail(addr, auth, m.cfg.SMTPFrom, []string{to}, []byte(msg.String()))
}

func (m *Mailer) SendToMany(emails []string, subject, htmlBody string) []error {
	var errs []error
	for _, email := range emails {
		if err := m.Send(email, subject, htmlBody); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", email, err))
		}
	}
	return errs
}
