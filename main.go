package main

import (
	"flag"

	"github.com/GalvinGao/connchk/cmd"
)

func main() {
	mode := flag.String("mode", "server", "server or sender")
	flag.Parse()

	switch *mode {
	case "server":
		cmd.StartServerMode()
	case "sender":
		cmd.StartSenderMode()
	default:
		panic("unknown mode")
	}
}
