package email

import (
	"bytes"
	"fmt"
	"html/template"

	"github.com/047pegasus/go-boilerplate/internal/config"
	"github.com/pkg/errors"
	"github.com/resend/resend-go/v4"
	"github.com/rs/zerolog"
)

type Client struct {
	client *resend.Client
	logger *zerolog.Logger
}

func NewClient(cfg *config.Config, logger *zerolog.Logger) *Client {
	return &Client{
		client: resend.NewClient(cfg.Integrations.ResendAPIKey),
		logger: logger,
	}
}

func (c *Client) SendEmail(to, subject string, templateName Template, data map[string]string) error {
	tmplPath := fmt.Sprintf("%s%s.html", "templates/email", templateName)

	tmpl, err := template.ParseFiles(tmplPath)
	if err != nil {
		return errors.Wrapf(err, "failed to parse email template %s at %s", templateName, tmplPath)
	}

	var body bytes.Buffer
	if err := tmpl.Execute(&body, data); err != nil {
		return errors.Wrapf(err, "failed to execute email template %s at %s", templateName, tmplPath)
	}
	params := &resend.SendEmailRequest{
		From:    fmt.Sprintf("%s <%s>", "Boilerplate", "onboarding@resend.dev"),
		To:      []string{to},
		Subject: subject,
		Html:    body.String(),
	}
	_, err = c.client.Emails.Send(params)
	if err != nil {
		return fmt.Errorf("failed to send email to %s with error: %w", to, err)
	}
	return nil
}
