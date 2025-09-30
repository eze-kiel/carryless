package email

import (
	"context"
	"fmt"
	"time"

	"carryless/internal/config"
	"carryless/internal/logger"
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
		// Set EU API base for European users
		if cfg.MailgunRegion == "EU" {
			client.SetAPIBase(mailgun.APIBaseEU)
		}
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

func (s *Service) SendWelcomeEmail(user *models.User, activationToken string) error {
	if !s.enabled {
		return fmt.Errorf("email service is not configured")
	}

	subject := fmt.Sprintf("Welcome to Carryless, %s! Please activate your account", user.Username)
	htmlBody := s.generateWelcomeHTML(user, activationToken)
	textBody := s.generateWelcomeText(user, activationToken)

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

	logger.Info("Welcome email sent",
		"email", user.Email,
		"user_id", user.ID,
		"message_id", resp)
	return nil
}

func (s *Service) SendAdminNotificationEmail(admin *models.User, newUser *models.User) error {
	if !s.enabled {
		return fmt.Errorf("email service is not configured")
	}

	subject := fmt.Sprintf("New User Registered - %s", newUser.Username)
	htmlBody := s.generateAdminNotificationHTML(admin, newUser)
	textBody := s.generateAdminNotificationText(admin, newUser)

	message := mailgun.NewMessage(
		s.domain,
		fmt.Sprintf("%s <%s>", s.senderName, s.senderEmail),
		subject,
		textBody,
		admin.Email,
	)
	message.SetHTML(htmlBody)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := s.client.Send(ctx, message)
	if err != nil {
		return fmt.Errorf("failed to send admin notification email to %s: %w", admin.Email, err)
	}

	logger.Info("Admin notification email sent",
		"admin_email", admin.Email,
		"admin_id", admin.ID,
		"new_user_email", newUser.Email,
		"new_user_id", newUser.ID,
		"message_id", resp)
	return nil
}