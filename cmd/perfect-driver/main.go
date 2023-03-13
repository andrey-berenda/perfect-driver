package main

import (
	"os"
	"os/signal"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/mymmrac/telego"

	"github.com/andrey-berenda/perfect-driver/internal/pkg/bot"
	"github.com/andrey-berenda/perfect-driver/internal/pkg/log"
	"github.com/andrey-berenda/perfect-driver/internal/pkg/storage"
)

func main() {
	time.Local = time.UTC
	logger := log.NewLogger()

	err := tgbotapi.SetLogger(log.NewBotLogger(logger))
	if err != nil {
		logger.Fatalf("tgbotapi.SetLogger: %v", err)
	}
	driverBot, err := telego.NewBot("6281856678:AAGQdSTnZwoU5SPjXsa8IKVVnbZmriqS-0c")
	if err != nil {
		logger.Fatalf("telego.NewBot: %v", err)
	}
	customerBot, err := telego.NewBot("6196016370:AAHuOP1C69M2hh9Z8DfhY9EImZgCUSq-OvY")
	if err != nil {
		logger.Fatalf("telego.NewBot: %v", err)
	}
	store := storage.New()

	b := bot.New(driverBot, customerBot, store, logger, -1001520856813)
	wg := &sync.WaitGroup{}

	wg.Add(1)
	go func() {
		updates, _ := driverBot.UpdatesViaLongPolling(&telego.GetUpdatesParams{
			Timeout: 10,
			AllowedUpdates: []string{
				"chat_join_request",
				"message",
				"callback_query",
				// "edited_message",
				"channel_post",
			},
		})

		logger.Info("Starting handle driver messages")
		b.HandleCustomerUpdates(updates)
		logger.Info("Handling messages driver stopped")
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		updates, _ := customerBot.UpdatesViaLongPolling(&telego.GetUpdatesParams{
			Timeout: 10,
			AllowedUpdates: []string{
				"chat_join_request",
				"message",
				"callback_query",
				"channel_post",
				// "edited_message",
				"channel_post",
			},
		})

		logger.Info("Starting handle customer messages")
		b.HandleCustomerUpdates(updates)
		logger.Info("Handling messages customer stopped")
		wg.Done()
	}()

	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, os.Interrupt)
	<-sigint
	driverBot.StopLongPolling()
	customerBot.StopLongPolling()
	wg.Wait()
	logger.Info("Bot gracefully stopped")
	_ = logger.Sync()
}
