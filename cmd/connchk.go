package cmd

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/twilio/twilio-go"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/GalvinGao/connchk/pkg/config"
	"github.com/GalvinGao/connchk/pkg/connchk"
	"github.com/GalvinGao/connchk/pkg/notify"
	"github.com/GalvinGao/connchk/pkg/subs"
)

func StartSenderMode() {
	conf, err := config.Parse()
	if err != nil {
		panic(err)
	}

	client := &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:        20,
			MaxIdleConnsPerHost: 20,
		},
		Timeout: conf.HeartbeatInterval,
	}

	for {
		log.Println("sending ping to", conf.PingToAddr)
		req, err := http.NewRequest("GET", conf.PingToAddr+"/ping", nil)
		if err != nil {
			log.Println("failed to create request:", err)
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), conf.HeartbeatInterval)
		req = req.WithContext(ctx)
		r, err := client.Do(req)
		if err != nil {
			log.Println("failed to send ping:", err)
		} else if r != nil && r.StatusCode != http.StatusOK {
			log.Println("unexpected status code while sending ping:", r.StatusCode)
		} else {
			log.Println("ping sent")
		}
		cancel()

		time.Sleep(conf.HeartbeatInterval)
	}
}

func StartServerMode() {
	conf, err := config.Parse()
	if err != nil {
		panic(err)
	}

	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(conf.MongoUri))
	if err != nil {
		panic(err)
	}

	subsvc := subs.New(client)

	twiliocli := twilio.NewRestClientWithParams(twilio.ClientParams{
		Username: conf.NotifyTwilioAccountSID,
		Password: conf.NotifyTwilioAuthToken,
	})

	notif, err := notify.New(conf, subsvc, twiliocli)
	if err != nil {
		panic(err)
	}

	ckr := connchk.New(conf.HeartbeatInterval, conf.GracePeriod)

	http.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		log.Println("received ping from", r.RemoteAddr)
		ckr.Ping()
	})

	http.HandleFunc("/subscribe", func(w http.ResponseWriter, r *http.Request) {
		log.Println("received subscription request from", r.RemoteAddr)

		channel := r.URL.Query().Get("channel")
		if channel != "sms" {
			log.Println("unknown channel:", channel)
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("unknown channel"))
			return
		}

		endpoint := r.URL.Query().Get("endpoint")
		if endpoint == "" || !strings.HasPrefix(endpoint, "+") {
			log.Println("invalid endpoint:", endpoint)
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("invalid endpoint"))
			return
		}

		if err := subsvc.Subscribe("sms", endpoint); err != nil {
			log.Println("failed to subscribe:", err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("failed to subscribe"))
			return
		}
	})

	http.HandleFunc("/unsubscribe", func(w http.ResponseWriter, r *http.Request) {
		log.Println("received unsubscription request from", r.RemoteAddr)

		channel := r.URL.Query().Get("channel")
		if channel != "sms" {
			log.Println("unknown channel:", channel)
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("unknown channel"))
			return
		}

		endpoint := r.URL.Query().Get("endpoint")
		if endpoint == "" || !strings.HasPrefix(endpoint, "+") {
			log.Println("invalid endpoint:", endpoint)
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("invalid endpoint"))
			return
		}

		if err := subsvc.Unsubscribe("sms", endpoint); err != nil {
			log.Println("failed to unsubscribe:", err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("failed to unsubscribe"))
			return
		}
	})

	http.HandleFunc("/callback/sms", func(w http.ResponseWriter, r *http.Request) {
		log.Println("received sms delivery report from", r.RemoteAddr)

		from := r.URL.Query().Get("From")
		body := r.URL.Query().Get("Body")
		if from == "" || body == "" {
			log.Println("invalid sms delivery report:", from, body)
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("invalid sms delivery report"))
			return
		}

		if strings.ToLower(strings.TrimSpace(body)) == "start" || strings.ToLower(strings.TrimSpace(body)) == "unstop" {
			if err := subsvc.Subscribe("sms", from); err != nil {
				log.Println("failed to subscribe:", err)
				return
			}
			log.Println("subscribed:", from)
		} else if strings.ToLower(strings.TrimSpace(body)) == "stop" {
			if err := subsvc.Unsubscribe("sms", from); err != nil {
				log.Println("failed to unsubscribe:", err)
				return
			}
			log.Println("unsubscribed:", from)
		}
	})

	go func() {
		http.ListenAndServe(conf.ServerListenAddr, nil)
	}()

	if !conf.NotificationDisabled {
		for {
			log.Println("checking status")
			if err := ckr.Check(); err != nil {
				log.Println("connection down:", err)
				notif.Down(time.Now(), err.Error())
			} else {
				log.Println("connection up")
				notif.Up()
			}

			time.Sleep(conf.HeartbeatInterval)
		}
	} else {
		log.Println("notification disabled")
		select {}
	}
}
