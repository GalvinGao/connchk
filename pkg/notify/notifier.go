package notify

import (
	"context"
	"log"
	"time"

	"github.com/nikoksr/notify"
	"github.com/nikoksr/notify/service/telegram"
	"github.com/twilio/twilio-go"
	openapi "github.com/twilio/twilio-go/rest/api/v2010"

	"github.com/GalvinGao/connchk/pkg/config"
	"github.com/GalvinGao/connchk/pkg/subs"
)

var timeFormat = "2006-01-02 15:04:05.999999 MST"

type NotifyI interface {
	// Notify that the connection is down, with the given error.
	Down(at time.Time, with string)
}

// Notifier only alerts when the connection is down for three or more consecutive checks
type Notifier struct {
	adminn     *notify.Notify
	twiliocli  *twilio.RestClient
	twiliosid  string
	twiliofrom string
	downCount  int

	alreadyDownAt *time.Time
	subsvc        *subs.Svc
}

func New(conf *config.Config, subsvc *subs.Svc, twiliocli *twilio.RestClient) (*Notifier, error) {
	n := notify.New()
	telegramService, err := telegram.New(conf.NotifyTelegramBotToken)
	if err != nil {
		return nil, err
	}
	telegramService.AddReceivers(int64(conf.NotifyTelegramReceiver))

	return &Notifier{
		adminn:     n,
		twiliocli:  twiliocli,
		twiliosid:  conf.NotifyTwilioAccountSID,
		twiliofrom: conf.NotifyTwilioFromPhone,
		subsvc:     subsvc,
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

	log.Println("sending notification that internet is down due to error:", with)

	go n.sendDown(at)
}

func (n *Notifier) sendDown(at time.Time) {
	subj := "[NETDOWN] Internet appears DOWN"
	msg := "Internet connection appears to be down since " + at.Format(timeFormat)

	go n.sendAdmin(msg)
	go n.sendActiveSubs(subj, msg)
}

func (n *Notifier) Up() {
	n.downCount = 0
	if n.alreadyDownAt == nil {
		return
	}
	alreadyDownAt := *n.alreadyDownAt
	n.alreadyDownAt = nil

	log.Println("sending notification that internet is up")

	go n.sendUp(alreadyDownAt)
}

func (n *Notifier) sendUp(alreadyDownAt time.Time) {
	subj := "[NETDOWN] Internet appears UP"
	msg := "It was down since " + alreadyDownAt.Format(timeFormat) + " and lasted for " + time.Since(alreadyDownAt).String()

	go n.sendAdmin(msg)
	go n.sendActiveSubs(subj, msg)
}

func (n *Notifier) sendAdmin(msg string) {
	err := n.adminn.Send(context.Background(), "[NETDOWN] Admin", msg)
	if err != nil {
		log.Println("failed to send notification:", err)
	}
}

func (n *Notifier) sendActiveSubs(subj, msg string) error {
	subs, err := n.subsvc.ListActiveSubs("sms")
	if err != nil {
		return err
	}

	m := subj + "\n" + msg

	for i, sub := range subs {
		r, err := n.twiliocli.Api.CreateMessage(&openapi.CreateMessageParams{
			Body:           &m,
			From:           &n.twiliofrom,
			To:             &sub.Endpoint,
			PathAccountSid: &n.twiliosid,
		})
		if err != nil {
			log.Println("failed to send notification:", err)
		} else {
			log.Println("sent notification to #", i, " (", sub.Endpoint, "):", r.Sid)
		}
	}

	return nil
}
