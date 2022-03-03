package server

import (
	log "github.com/sirupsen/logrus"
	"os"
	"os/signal"
	"syscall"
)

func Start(options Options) {
	// init webhook api
	ws, err := NewWebhookServer(options)
	if err != nil {
		panic(err)
	}

	// start webhook server in new routine
	go ws.Start()
	log.Info("Server started")

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	<-signalChan

	ws.Stop()
}
