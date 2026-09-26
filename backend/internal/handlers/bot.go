package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/url"
	"strings"
	"time"

	"mtproxy-manager/internal/config"
	"mtproxy-manager/internal/database"
	"mtproxy-manager/internal/models"
)

// Bot sends Telegram Bot API requests and the service's messages to users.
type Bot struct {
	cfg *config.Config
	db  *database.DB
}

func NewBot(cfg *config.Config, db *database.DB) *Bot {
	return &Bot{cfg: cfg, db: db}
}

func (b *Bot) Enabled() bool { return b.cfg.TelegramBotToken != "" }

// Call invokes a Bot API method and decodes "result" into out (may be nil).
func (b *Bot) Call(method string, params map[string]any, out any) error {
	if !b.Enabled() {
		return fmt.Errorf("telegram bot token is not configured")
	}
	body, _ := json.Marshal(params)
	resp, err := httpClient.Post("https://api.telegram.org/bot"+b.cfg.TelegramBotToken+"/"+method, "application/json", bytes.NewReader(body))
	if err != nil {
		var ue *url.Error
		if errors.As(err, &ue) {
			err = ue.Err // the URL contains the bot token; keep it out of logs
		}
		return fmt.Errorf("%s: %w", method, err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	var envelope struct {
		OK          bool            `json:"ok"`
		Description string          `json:"description"`
		Result      json.RawMessage `json:"result"`
	}
	if err := json.Unmarshal(respBody, &envelope); err != nil {
		return fmt.Errorf("%s: HTTP %d: %s", method, resp.StatusCode, truncateBody(respBody))
	}
	if !envelope.OK {
		return fmt.Errorf("%s: %s", method, envelope.Description)
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(envelope.Result, out)
}

func (b *Bot) send(chatID int64, text string) error {
	return b.Call("sendMessage", map[string]any{
		"chat_id":                  chatID,
		"text":                     text,
		"disable_web_page_preview": true,
		"reply_markup":             b.openAppButton(),
	}, nil)
}

// openAppButton opens the Mini App directly when BASE_URL is https, else TG_PAY_URL.
func (b *Bot) openAppButton() map[string]any {
	button := map[string]any{"text": "Открыть Stay", "url": b.cfg.TelegramPayURL}
	if base := strings.TrimRight(b.cfg.BaseURL, "/"); strings.HasPrefix(base, "https://") {
		button = map[string]any{"text": "Открыть Stay", "web_app": map[string]string{"url": base}}
	}
	return map[string]any{"inline_keyboard": [][]map[string]any{{button}}}
}

func (b *Bot) SendWelcome(chatID int64) error {
	return b.send(chatID, `👋 Добро пожаловать в Stay!

🛡️ Быстрое и стабильное защищённое подключение прямо из Telegram.

💪 Мгновенная настройка • Высокоскоростные серверы • Работает на всех устройствах`)
}

func (b *Bot) NotifySubscriptionPaid(user *models.User, plan *models.Plan) {
	if !b.Enabled() || user.TelegramID == 0 {
		return
	}
	text := fmt.Sprintf("✅ Успешная оплата!\n\nПодписка Stay на %s активирована.\nОткройте приложение, чтобы настроить подключение.", plan.Name)
	if err := b.send(user.TelegramID, text); err != nil {
		log.Printf("notify paid user=%d: %v", user.ID, err)
	}
}

// RunExpiryReminders DMs users whose subscription ends in 7 or 1 UTC calendar
// days. Flags on the subscription make it safe to run repeatedly.
func (b *Bot) RunExpiryReminders() {
	if !b.Enabled() {
		return
	}
	for _, r := range []struct {
		days int
		is7d bool
		mark func(int64) error
		text string
	}{
		{7, true, b.db.MarkSubscriptionExpiryReminder7dSent, "⏳ Подписка Stay (%s) закончится через 7 дней.\n\nДоступ активен до: %s\n\nПродлите подписку, чтобы не потерять доступ."},
		{1, false, b.db.MarkSubscriptionExpiryReminder1dSent, "⚠️ Подписка Stay (%s) заканчивается через 1 день.\n\nДоступ активен до: %s\n\nПродлите подписку, чтобы не потерять доступ."},
	} {
		rows, err := b.db.ListSubscriptionsExpiryCalendarDays(r.days, r.is7d)
		if err != nil {
			log.Printf("expiry reminders %dd: %v", r.days, err)
			continue
		}
		for _, row := range rows {
			planName := row.PlanID
			if plan := models.GetPlan(row.PlanID); plan != nil {
				planName = plan.Name
			}
			if err := b.send(row.TelegramID, fmt.Sprintf(r.text, planName, moscowTime(row.ExpiresAt))); err != nil {
				log.Printf("expiry reminder sub=%d: %v", row.ID, err)
				continue
			}
			if err := r.mark(row.ID); err != nil {
				log.Printf("expiry reminder mark sub=%d: %v", row.ID, err)
			}
		}
	}
}

var moscow = func() *time.Location {
	if loc, err := time.LoadLocation("Europe/Moscow"); err == nil {
		return loc
	}
	return time.FixedZone("MSK", 3*60*60)
}()

func moscowTime(t time.Time) string {
	return t.In(moscow).Format("02.01.2006 15:04") + " (МСК)"
}
