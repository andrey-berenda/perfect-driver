package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/mymmrac/telego"

	"github.com/andrey-berenda/perfect-driver/internal/pkg/storage"
)

type Handler struct {
}

type OrderData struct {
	Source      string
	Destination string
	Time        string
	Phone       string
}

func Parse(payload string) (OrderData, error) {
	b, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return OrderData{}, fmt.Errorf("url.Parse: %w", err)
	}
	u, err := url.ParseQuery(string(b))
	if err != nil {
		return OrderData{}, fmt.Errorf("url.ParseQuery: %w", err)
	}
	return OrderData{
		Phone:       u.Get("Phone"),
		Time:        u.Get("Time"),
		Source:      u.Get("Source"),
		Destination: u.Get("Destination"),
	}, nil
}

type Data struct {
	Body string `json:"body"`
}

func (h Handler) Invoke(ctx context.Context, payload []byte) ([]byte, error) {

	data := Data{}
	err := json.Unmarshal(payload, &data)
	if err != nil {
		return nil, fmt.Errorf("json.Unmarshal: %w", err)
	}
	store := storage.New()
	o, err := Parse(data.Body)
	if err != nil {
		sendMessage(data.Body, 0)
		return nil, fmt.Errorf("parse: %w", err)
	}

	order, err := store.OrderCreateFromLambda(ctx, o.Source, o.Destination, o.Time, o.Phone)
	if err != nil {
		panic(err)
	}
	sendMessage(order.ToDriverChat(), order.ID)
	return nil, err
}

func sendMessage(text string, orderID int) {
	driverBot, err := telego.NewBot("6281856678:AAGQdSTnZwoU5SPjXsa8IKVVnbZmriqS-0c")
	if err != nil {
		panic(err)
	}
	_, err = driverBot.SendMessage(&telego.SendMessageParams{
		ChatID: telego.ChatID{ID: -1001520856813},
		Text:   text,
		ReplyMarkup: &telego.InlineKeyboardMarkup{
			InlineKeyboard: [][]telego.InlineKeyboardButton{
				{
					{Text: "Возьму заказ", CallbackData: fmt.Sprintf("%s:%d", "order", orderID)},
				},
			},
		},
	})
	if err != nil {
		panic(err)
	}
}

func main() {
	var h lambda.Handler = Handler{}
	lambda.Start(h)
}
