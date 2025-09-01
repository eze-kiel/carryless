package email

import (
	"context"
	"fmt"
	"log"
	"time"

	"carryless/internal/config"
	"carryless/internal/models"

	"github.com/mailgun/mailgun-go/v5"
)

type Service struct {
	client      mailgun.Mailgun
	domain      string
	senderEmail string
	senderName  string
	enabled     bool
}

func NewService(cfg *config.Config) *Service {
	enabled := cfg.MailgunDomain != "" && cfg.MailgunAPIKey != ""
	
	var client mailgun.Mailgun
	if enabled {
		client = mailgun.NewMailgun(cfg.MailgunAPIKey)
	}
	
	return &Service{
		client:      client,
		domain:      cfg.MailgunDomain,
		senderEmail: cfg.MailgunSenderEmail,
		senderName:  cfg.MailgunSenderName,
		enabled:     enabled,
	}
}

func (s *Service) IsEnabled() bool {
	return s.enabled
}

func (s *Service) SendWelcomeEmail(user *models.User) error {
	if !s.enabled {
		return fmt.Errorf("email service is not configured")
	}

	subject := fmt.Sprintf("Welcome to Carryless, %s!", user.Username)
	htmlBody := s.generateWelcomeHTML(user)
	textBody := s.generateWelcomeText(user)

	message := mailgun.NewMessage(
		s.domain,
		fmt.Sprintf("%s <%s>", s.senderName, s.senderEmail),
		subject,
		textBody,
		user.Email,
	)
	message.SetHTML(htmlBody)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := s.client.Send(ctx, message)
	if err != nil {
		return fmt.Errorf("failed to send welcome email to %s: %w", user.Email, err)
	}

	log.Printf("Welcome email sent to %s (Message ID: %s)", user.Email, resp)
	return nil
}