// Package email provides SMTP-based email sending for Svenskt Vin.
package email

import (
	"fmt"
	"net/smtp"
	"strings"
)

// Config holds SMTP server settings.
type Config struct {
	Host string
	Port int
	User string
	Pass string
	From string
}

// Sender wraps SMTP client for sending emails.
type Sender struct {
	cfg Config
}

// NewSender creates a new email sender.
func NewSender(cfg Config) *Sender {
	return &Sender{cfg: cfg}
}

// sendMail sends an email via SMTP.
func (s *Sender) sendMail(to, subject, body string) error {
	addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)
	auth := smtp.PlainAuth("", s.cfg.User, s.cfg.Pass, s.cfg.Host)

	msg := fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s",
		s.cfg.From, to, subject, body,
	)

	err := smtp.SendMail(addr, auth, s.cfg.From, []string{to}, []byte(msg))
	if err != nil {
		return fmt.Errorf("send email: %w", err)
	}
	return nil
}

// SendMagicLink sends a magic link login email.
func (s *Sender) SendMagicLink(email, token string) error {
	basePath := "/auth/verify?token="
	link := basePath + token
	subject := "Din inloggningslänk"
	body := `
		<html>
		<body>
		<h1>Inloggning — Svenskt Vin</h1>
		<p>Klicka på länken nedan för att logga in:</p>
		<p>
			<a href="` + link + `" style="background-color: #2d6a2d; color: white; padding: 14px 20px; text-align: center; text-decoration: none; display: inline-block; border-radius: 4px;">
				Logga in
			</a>
		</p>
		<p>Om du inte begärde detta, ignorera detta e-postmeddelande.</p>
		<p>Denna länk är endast giltig i 15 minuter.</p>
		<p>Mvh, <strong>Svenskt Vin</strong></p>
		</body>
		</html>
	`
	return s.sendMail(email, subject, body)
}

// SendPasswordReset sends a password reset email.
func (s *Sender) SendPasswordReset(email, token string) error {
	basePath := "/auth/set-password?token="
	link := basePath + token
	subject := "Återställ ditt lösenord"
	body := `
		<html>
		<body>
		<h1>Återställ ditt lösenord</h1>
		<p>Klicka på länken nedan för att återställa ditt lösenord:</p>
		<p>
			<a href="` + link + `" style="background-color: #4CAF50; color: white; padding: 14px 20px; text-align: center; text-decoration: none; display: inline-block; border-radius: 4px;">
				Återställ lösenord
			</a>
		</p>
		<p>Om du inte begärde detta, ignorera detta e-postmeddelande.</p>
		<p>Denna länk är endast giltig i 1 timme.</p>
		<p>Mvh, <strong>Svenskt Vin</strong></p>
		</body>
		</html>
	`
	return s.sendMail(email, subject, body)
}

// SendInvite sends a vineyard membership invite email.
func (s *Sender) SendInvite(appHost, vineyardName, token string) error {
	link := appHost + "/invite?token=" + token
	subject := fmt.Sprintf("Inbjudan till %s", vineyardName)
	body := `
		<html>
		<body>
		<h1>Inbjudan till ` + vineyardName + `</h1>
		<p>Du har blivit inbjuden att gå med i <strong>` + vineyardName + `</strong> på Svenskt Vin.</p>
		<p>
			<a href="` + link + `" style="background-color: #2d6a2d; color: white; padding: 14px 20px; text-align: center; text-decoration: none; display: inline-block; border-radius: 4px;">
				Acceptera inbjudan
			</a>
		</p>
		<p>Den här länken är giltig i 7 dagar.</p>
		<p>Mvh, <strong>Svenskt Vin</strong></p>
		</body>
		</html>
	`
	return s.sendMail(token, subject, body) // NOTE: This is a placeholder - the actual email is passed from the handler
}

// SendInviteWithEmail sends a vineyard membership invite email to a specific address.
func (s *Sender) SendInviteWithEmail(toEmail, appHost, vineyardName, token string) error {
	link := appHost + "/invite?token=" + token
	subject := fmt.Sprintf("Inbjudan till %s", vineyardName)
	body := `
		<html>
		<body>
		<h1>Inbjudan till ` + vineyardName + `</h1>
		<p>Du har blivit inbjuden att gå med i <strong>` + vineyardName + `</strong> på Svenskt Vin.</p>
		<p>
			<a href="` + link + `" style="background-color: #2d6a2d; color: white; padding: 14px 20px; text-align: center; text-decoration: none; display: inline-block; border-radius: 4px;">
				Acceptera inbjudan
			</a>
		</p>
		<p>Den här länken är giltig i 7 dagar.</p>
		<p>Mvh, <strong>Svenskt Vin</strong></p>
		</body>
		</html>
	`
	return s.sendMail(toEmail, subject, body)
}

// sanitizeHTML removes any potentially dangerous HTML from user input.
func sanitizeHTML(s string) string {
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&#x27;")
	return s
}
