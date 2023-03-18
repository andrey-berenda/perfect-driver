package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/mymmrac/telego"

	"github.com/andrey-berenda/perfect-driver/internal/pkg/bot"
	"github.com/andrey-berenda/perfect-driver/internal/pkg/log"
	"github.com/andrey-berenda/perfect-driver/internal/pkg/processing"
	"github.com/andrey-berenda/perfect-driver/internal/pkg/storage"
)

func main() {
	time.Local = time.UTC
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)

	logger := log.NewLogger()
	driverBot, err := telego.NewBot("6281856678:AAGQdSTnZwoU5SPjXsa8IKVVnbZmriqS-0c")
	if err != nil {
		logger.Fatalf("telego.NewBot: %v", err)
	}
	customerBot, err := telego.NewBot("6196016370:AAHuOP1C69M2hh9Z8DfhY9EImZgCUSq-OvY")
	if err != nil {
		logger.Fatalf("telego.NewBot: %v", err)
	}
	store := storage.New(logger)

	yookassaPaymentsURL := "https://api.yookassa.ru/v3/payments"

	processor := processing.New(
		store,
		http.DefaultClient,
		base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf(
			"%d:%s",
			974352,
			"test_IFouxng-0oLRTgFPHEVMB5knuEPEvPL5siwYcSKf3pA",
		))),
		yookassaPaymentsURL,
		func(orderID uuid.UUID) string {
			return fmt.Sprintf("%s/%s", yookassaPaymentsURL, orderID.String())
		},
	)

	b := bot.New(driverBot, customerBot, processor, store, logger, -1001520856813)
	wg := &sync.WaitGroup{}

	wg.Add(1)
	go func() {
		updates, err := driverBot.UpdatesViaLongPolling(&telego.GetUpdatesParams{
			Timeout: 10,
			AllowedUpdates: []string{
				telego.MessageUpdates,
				telego.CallbackQueryUpdates,
				telego.ChannelPostUpdates,
			},
		})
		if err != nil {
			cancel()
			logger.Errorf("driverBot.UpdatesViaLongPolling: %v", err)
			return
		}
		logger.Info("Starting handle driver messages")
		b.HandleCustomerUpdates(updates)
		logger.Info("Handling messages driver stopped")
		wg.Done()

	}()

	wg.Add(1)
	go func() {
		updates, err := customerBot.UpdatesViaLongPolling(&telego.GetUpdatesParams{
			Timeout: 10,
			AllowedUpdates: []string{
				telego.MessageUpdates,
				telego.CallbackQueryUpdates,
				telego.ChannelPostUpdates,
			},
		})
		if err != nil {
			cancel()
			logger.Errorf("driverBot.UpdatesViaLongPolling: %v", err)
			return
		}

		logger.Info("Starting handle customer messages")
		b.HandleCustomerUpdates(updates)
		logger.Info("Handling messages customer stopped")
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		logger.Info("Starting checking payments")
		b.CheckPayments(ctx, store.PaymentsForCheckChan(ctx))
		logger.Info("Checking payments stopped")
		wg.Done()
	}()

	<-ctx.Done()

	driverBot.StopLongPolling()
	customerBot.StopLongPolling()
	wg.Wait()
	logger.Info("Bot gracefully stopped")
	_ = logger.Sync()
}
