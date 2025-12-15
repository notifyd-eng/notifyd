package notify

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"

	"github.com/notifyd-eng/notifyd/internal/config"
	"github.com/notifyd-eng/notifyd/internal/store"
)

type EmailSender struct {
	cfg config.EmailConfig
}

func NewEmailSender(cfg config.EmailConfig) *EmailSender {
	return &EmailSender{cfg: cfg}
}

func (e *EmailSender) Channel() string { return "email" }

func (e *EmailSender) Send(ctx context.Context, n *store.Notification) error {
	addr := fmt.Sprintf("%s:%d", e.cfg.Host, e.cfg.Port)

	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=utf-8\r\n\r\n%s",
		e.cfg.From, n.Recipient, n.Subject, n.Body)

	var auth smtp.Auth
	if e.cfg.Username != "" {
		auth = smtp.PlainAuth("", e.cfg.Username, e.cfg.Password, e.cfg.Host)
	}

	if e.cfg.UseTLS {
		return e.sendTLS(addr, auth, msg, n.Recipient)
	}

	return smtp.SendMail(addr, auth, e.cfg.From, []string{n.Recipient}, []byte(msg))
}

func (e *EmailSender) sendTLS(addr string, auth smtp.Auth, msg, recipient string) error {
	host, _, _ := net.SplitHostPort(addr)

	conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: host})
	if err != nil {
		return fmt.Errorf("tls dial: %w", err)
	}

	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return fmt.Errorf("smtp client: %w", err)
	}
	defer client.Close()

	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("auth: %w", err)
		}
	}

	if err := client.Mail(e.cfg.From); err != nil {
		return err
	}
	if err := client.Rcpt(recipient); err != nil {
		return err
	}

	w, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write([]byte(msg)); err != nil {
		return err
	}

	return w.Close()
}
