package registrationnotify

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/terrynullson/mntrng/internal/domain"
	"github.com/terrynullson/mntrng/internal/telegram"
)

type TelegramNotifier struct {
	telegramClient *telegram.Client
	botToken       string
	chatID         string
	timeout        time.Duration
}

func NewTelegramNotifier(telegramClient *telegram.Client, botToken string, chatID string) *TelegramNotifier {
	return &TelegramNotifier{
		telegramClient: telegramClient,
		botToken:       strings.TrimSpace(botToken),
		chatID:         strings.TrimSpace(chatID),
		timeout:        5 * time.Second,
	}
}

func (n *TelegramNotifier) NotifyNewRegistrationRequest(ctx context.Context, request domain.RegistrationRequest) error {
	if n == nil || n.telegramClient == nil || n.botToken == "" || n.chatID == "" {
		return nil
	}

	notifyCtx, cancel := context.WithTimeout(ctx, n.timeout)
	defer cancel()

	text := fmt.Sprintf(
		"New registration request #%d\ncompany_id=%d\nemail=%s\nlogin=%s\nrequested_role=%s",
		request.ID,
		request.CompanyID,
		request.Email,
		request.Login,
		request.RequestedRole,
	)

	err := n.telegramClient.SendMessage(notifyCtx, n.botToken, n.chatID, text)
	if err != nil {
		log.Printf("registration notify failed: request_id=%d company_id=%d err=%v", request.ID, request.CompanyID, err)
	}
	return err
}
