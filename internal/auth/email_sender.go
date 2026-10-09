package auth

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"mime"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"net/smtp"
	"time"
)

type SMTPSettings struct{ Host, Port, Username, Password, From, Security string }
type SMTPSender struct{ settings SMTPSettings }

func NewSMTPSender(settings SMTPSettings) *SMTPSender { return &SMTPSender{settings: settings} }
func (s *SMTPSender) Ready() bool {
	return s.settings.Host != "" && s.settings.Port != "" && s.settings.Username != "" && s.settings.Password != "" && s.settings.From != ""
}
func (s *SMTPSender) SendCode(ctx context.Context, email, code string) error {
	return s.send(ctx, email, "Код подтверждения Grace", fmt.Sprintf("Ваш код подтверждения Grace: %s\r\n\r\nВведите его в той же вкладке сайта. Код действует 10 минут.\r\nЕсли вы не запрашивали код, проигнорируйте это письмо.\r\n", code))
}
func (s *SMTPSender) SendLink(ctx context.Context, email, link, purpose string) error {
	subject := "Восстановление пароля Grace"
	if purpose == "register" {
		subject = "Подтверждение регистрации Grace"
	}
	return s.send(ctx, email, subject, fmt.Sprintf("%s\r\n\r\nОткройте ссылку и подтвердите действие на сайте:\r\n%s\r\n\r\nСсылка действует 30 минут и может быть использована один раз.\r\nЕсли вы не запрашивали действие, проигнорируйте письмо.\r\n", subject, link))
}
func (s *SMTPSender) send(ctx context.Context, email, subject, text string) error {
	if !s.Ready() {
		return fmt.Errorf("mail not configured")
	}
	from, err := mail.ParseAddress(s.settings.From)
	if err != nil {
		return fmt.Errorf("invalid sender")
	}
	address := net.JoinHostPort(s.settings.Host, s.settings.Port)
	dialer := net.Dialer{Timeout: 10 * time.Second}
	connection, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return fmt.Errorf("mail connection unavailable")
	}
	defer connection.Close()
	_ = connection.SetDeadline(time.Now().Add(15 * time.Second))
	config := &tls.Config{ServerName: s.settings.Host, MinVersion: tls.VersionTLS12}
	if s.settings.Security == "tls" {
		secure := tls.Client(connection, config)
		if err = secure.HandshakeContext(ctx); err != nil {
			return fmt.Errorf("mail TLS unavailable")
		}
		connection = secure
	}
	client, err := smtp.NewClient(connection, s.settings.Host)
	if err != nil {
		return fmt.Errorf("mail protocol unavailable")
	}
	defer client.Close()
	if s.settings.Security == "starttls" {
		if ok, _ := client.Extension("STARTTLS"); !ok {
			return fmt.Errorf("mail TLS required")
		}
		if err = client.StartTLS(config); err != nil {
			return fmt.Errorf("mail TLS unavailable")
		}
	} else if s.settings.Security != "tls" {
		return fmt.Errorf("mail TLS required")
	}
	if err = client.Auth(smtp.PlainAuth("", s.settings.Username, s.settings.Password, s.settings.Host)); err != nil {
		return fmt.Errorf("mail authentication failed")
	}
	if err = client.Mail(from.Address); err != nil {
		return fmt.Errorf("mail sender rejected")
	}
	if err = client.Rcpt(email); err != nil {
		return fmt.Errorf("mail recipient rejected")
	}
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("mail send failed")
	}
	var message bytes.Buffer
	fmt.Fprintf(&message, "From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\nContent-Transfer-Encoding: quoted-printable\r\n\r\n", from.String(), (&mail.Address{Address: email}).String(), mime.BEncoding.Encode("UTF-8", subject))
	body := quotedprintable.NewWriter(&message)
	_, _ = body.Write([]byte(text))
	_ = body.Close()
	if _, err = writer.Write(message.Bytes()); err != nil {
		return fmt.Errorf("mail send failed")
	}
	if err = writer.Close(); err != nil {
		return fmt.Errorf("mail send failed")
	}
	return client.Quit()
}
