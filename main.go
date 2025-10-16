package main

import (
	"clapper/bot"
	"clapper/config"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	cfg := config.Load()

	movieBot, err := bot.New(cfg)
	if err != nil {
		log.Fatal("Error creating bot:", err)
	}

	if err := movieBot.Start(); err != nil {
		log.Fatal("Error starting bot:", err)
	}

	log.Println("Bot is running. Press CTRL+C to exit.")
	
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	log.Println("Shutting down...")
	movieBot.Stop()
}