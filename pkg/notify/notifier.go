package notify

import (
	"context"
	"log"
	"time"

	"github.com/nikoksr/notify"
	"github.com/nikoksr/notify/service/telegram"
	"github.com/nikoksr/notify/service/twilio"

	"github.com/GalvinGao/connchk/pkg/config"
)

var timeFormat = "2006-01-02 15:04:05.999999 MST"

type NotifyI interface {
	// Notify that the connection is down, with the given error.
	Down(at time.Time, with string)
}

// Notifier only alerts when the connection is down for three or more consecutive checks
type Notifier struct {
	n *notify.Notify

	alreadyDownAt *time.Time
	downCount     int
}

func New(conf *config.Config) (*Notifier, error) {
	n := notify.New()
	telegramService, err := telegram.New(conf.NotifyTelegramBotToken)
	if err != nil {
		return nil, err
	}
	telegramService.AddReceivers(int64(conf.NotifyTelegramReceiver))

	twilioService, err := twilio.New(conf.NotifyTwilioAccountSID, conf.NotifyTwilioAuthToken, conf.NotifyTwilioFromPhone)
	if err != nil {
		return nil, err
	}
	twilioService.AddReceivers(conf.NotifyTwilioToPhone)

	n.UseServices(telegramService, twilioService)

	return &Notifier{
		n: n,
	}, nil
}

func (n *Notifier) Down(at time.Time, with string) {
	n.downCount++
	if n.downCount < 3 {
		return
	}

	if n.alreadyDownAt != nil {
		return
	}
	n.alreadyDownAt = &at

	go func() {
		err := n.n.Send(context.Background(), "[FMNETCHK] Internet appears DOWN", "Internet connection appears to be down since "+at.Format(timeFormat))
		if err != nil {
			log.Println("failed to send notification:", err)
		}
	}()
}

func (n *Notifier) Up() {
	n.downCount = 0

	if n.alreadyDownAt == nil {
		return
	}
	alreadyDownAt := *n.alreadyDownAt
	n.alreadyDownAt = nil

	go func() {
		err := n.n.Send(context.Background(), "[FMNETCHK] Internet appears UP", "It was down since "+alreadyDownAt.Format(timeFormat)+" and lasted for "+time.Since(alreadyDownAt).String())
		if err != nil {
			log.Println("failed to send notification:", err)
		}
	}()
}
