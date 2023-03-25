package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/mymmrac/telego"

	"github.com/andrey-berenda/perfect-driver/internal/pkg/log"
	"github.com/andrey-berenda/perfect-driver/internal/pkg/storage"
)

type Handler struct {
}

type OrderData struct {
	Source      string
	Destination string
	Time        string
	Phone       string
	Name        string
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
		Name:        u.Get("Name"),
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
	store := storage.New(log.NewLogger())
	o, err := Parse(data.Body)
	if err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}
	bot, err := telego.NewBot("6281856678:AAGQdSTnZwoU5SPjXsa8IKVVnbZmriqS-0c")
	if err != nil {
		panic(err)
	}

	if o.Name != "" {
		_, err = bot.SendMessage(&telego.SendMessageParams{
			ChatID: telego.ChatID{ID: -908250161},
			Text: fmt.Sprintf(`Новая заявка от водителя
Телефон: %s
Имя: %s
`, o.Phone, o.Name),
		})
		return nil, err
	}
	order, err := store.OrderCreateFromLambda(ctx, o.Source, o.Destination, o.Time, o.Phone)
	if err != nil {
		panic(err)
	}
	sendMessage(order.ToDriverChat(), order.ID, bot)
	return nil, err
}

func sendMessage(text string, orderID int, bot *telego.Bot) {
	_, err := bot.SendMessage(&telego.SendMessageParams{
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
