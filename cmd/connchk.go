package cmd

import (
	"log"
	"net/http"
	"time"

	"github.com/GalvinGao/connchk/pkg/config"
	"github.com/GalvinGao/connchk/pkg/connchk"
	"github.com/GalvinGao/connchk/pkg/notify"
)

func StartSenderMode() {
	conf, err := config.Parse()
	if err != nil {
		panic(err)
	}

	client := &http.Client{
		Timeout: conf.HeartbeatInterval,
	}

	for {
		log.Println("sending ping to", conf.PingToAddr)
		r, err := client.Get(conf.PingToAddr + "/ping")
		if err != nil {
			log.Println("failed to send ping:", err)
		} else if r != nil && r.StatusCode != http.StatusOK {
			log.Println("unexpected status code while sending ping:", r.StatusCode)
		} else {
			log.Println("ping sent")
		}

		time.Sleep(conf.HeartbeatInterval)
	}
}

func StartServerMode() {
	conf, err := config.Parse()
	if err != nil {
		panic(err)
	}

	notif, err := notify.New(conf)
	if err != nil {
		panic(err)
	}

	ckr := connchk.New(conf.HeartbeatInterval, conf.GracePeriod)

	http.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		log.Println("received ping from", r.RemoteAddr)
		ckr.Ping()
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
