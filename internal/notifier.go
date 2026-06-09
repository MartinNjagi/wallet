package internal

import "context"

// Notifier is the interface — add Email, Push, Slack etc. later
type Notifier interface {
	Send(ctx context.Context, to, message string) error
}

// SMSNotifier — thin wrapper, no auth logic
type SMSNotifier struct {
	client   *InternalClient
	senderID string
}

func NewSMSNotifier(client *InternalClient, senderID string) *SMSNotifier {
	return &SMSNotifier{client: client, senderID: senderID}
}

func (n *SMSNotifier) Send(ctx context.Context, to, message string) error {
	return n.client.Post(ctx, "sms", "/api/v1/internal/sms/send", map[string]any{
		"msisdn":    to,
		"message":   message,
		"sender_id": n.senderID,
	}, nil)
}

// EmailNotifier — stub, wire up when ready
type EmailNotifier struct {
	client *InternalClient
}

func NewEmailNotifier(client *InternalClient) *EmailNotifier {
	return &EmailNotifier{client: client}
}

func (n *EmailNotifier) Send(ctx context.Context, to, message string) error {
	return n.client.Post(ctx, "email", "/api/v1/internal/email/send", map[string]any{
		"to":      to,
		"message": message,
	}, nil)
}
