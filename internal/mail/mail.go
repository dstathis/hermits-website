package mail

import (
	"crypto/tls"
	"fmt"
	"net"
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

	return m.sendMail(to, subject, htmlBody)
}

func (m *Mailer) sendMail(to, subject, htmlBody string) error {
	addr := fmt.Sprintf("%s:%d", m.cfg.SMTPHost, m.cfg.SMTPPort)
	auth := smtp.PlainAuth("", m.cfg.SMTPUser, m.cfg.SMTPPass, m.cfg.SMTPHost)

	msg := buildMessage(m.cfg.SMTPFrom, to, subject, htmlBody)

	// Port 465 = implicit TLS (SMTPS), everything else = STARTTLS
	if m.cfg.SMTPPort == 465 {
		return m.sendImplicitTLS(addr, auth, to, msg)
	}
	return smtp.SendMail(addr, auth, m.cfg.SMTPFrom, []string{to}, msg)
}

func (m *Mailer) sendImplicitTLS(addr string, auth smtp.Auth, to string, msg []byte) error {
	host, _, _ := net.SplitHostPort(addr)
	tlsConfig := &tls.Config{ServerName: host}

	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("TLS dial: %w", err)
	}

	client, err := smtp.NewClient(conn, host)
	if err != nil {
		conn.Close()
		return fmt.Errorf("SMTP client: %w", err)
	}
	defer client.Close()

	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("SMTP auth: %w", err)
	}
	if err := client.Mail(m.cfg.SMTPFrom); err != nil {
		return fmt.Errorf("SMTP MAIL: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("SMTP RCPT: %w", err)
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("SMTP DATA: %w", err)
	}
	if _, err := w.Write(msg); err != nil {
		return fmt.Errorf("SMTP write: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("SMTP close data: %w", err)
	}
	return client.Quit()
}

func buildMessage(from, to, subject, htmlBody string) []byte {
	headers := map[string]string{
		"From":         from,
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
	return []byte(msg.String())
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
