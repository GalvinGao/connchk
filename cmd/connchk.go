package cmd

import (
	"log"
	"time"

	"github.com/GalvinGao/connchk/pkg/config"
	"github.com/GalvinGao/connchk/pkg/connchk"
	"github.com/GalvinGao/connchk/pkg/notify"
)

func Start() {
	conf, err := config.Parse()
	if err != nil {
		panic(err)
	}

	notif, err := notify.New(conf)
	if err != nil {
		panic(err)
	}

	ckr := connchk.New(conf.CheckAddr)

	for {
		log.Println("checking connection")
		now := time.Now()
		if err := ckr.Do(); err != nil {
			log.Println("connection down:", err)
			notif.Down(now, err.Error())
		} else {
			latency := time.Since(now)
			log.Println("connection up, latency:", latency)
			notif.Up()
		}

		time.Sleep(conf.CheckInterval)
	}
}
