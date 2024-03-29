package bot

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/mymmrac/telego"
	"github.com/mymmrac/telego/telegoutil"
	"go.uber.org/zap"

	"github.com/andrey-berenda/perfect-driver/internal/pkg/log"
	"github.com/andrey-berenda/perfect-driver/internal/pkg/models"
	"github.com/andrey-berenda/perfect-driver/internal/pkg/processing"
	"github.com/andrey-berenda/perfect-driver/internal/pkg/storage"
)

const createOrderCallback = "createOrder"
const arrivedCallback = "arrived"
const finishedCallback = "finished"
const orderCallback = "order"

type MessageHandler func(ctx context.Context, update telego.Update) bool
type CallbackHandler func(ctx context.Context, callback telego.CallbackQuery)

type Bot struct {
	driverBot        *telego.Bot
	customerBot      *telego.Bot
	processor        processing.Processor
	store            *storage.Store
	driversChatID    int64
	handlers         []MessageHandler
	commandHandlers  map[string]MessageHandler
	callbackHandlers map[string]CallbackHandler
	logger           *zap.SugaredLogger
}

func New(
	driverBot *telego.Bot,
	customerBot *telego.Bot,
	processor processing.Processor,
	store *storage.Store,
	logger *zap.SugaredLogger,
	driversChatID int64,
) *Bot {
	b := &Bot{
		driverBot:     driverBot,
		customerBot:   customerBot,
		processor:     processor,
		store:         store,
		logger:        logger,
		driversChatID: driversChatID,
	}
	b.handlers = []MessageHandler{
		b.HandleMessage,
	}
	b.commandHandlers = map[string]MessageHandler{
		"start": b.HandleStartCommand,
	}
	b.callbackHandlers = map[string]CallbackHandler{
		createOrderCallback: b.HandleCreateOrder,
		orderCallback:       b.HandleGetOrder,
		arrivedCallback:     b.HandleArrived,
		finishedCallback:    b.HandleFinished,
	}
	return b
}

func (b *Bot) HandleCustomerUpdates(updates <-chan telego.Update) {
	for update := range updates {
		ctx := context.Background()
		for _, handler := range b.handlers {
			if handler(ctx, update) {
				break
			}
		}
	}
}

func (b *Bot) HandleDriverUpdates(updates <-chan telego.Update) {
	for update := range updates {
		ctx := context.Background()
		for _, handler := range b.handlers {
			if handler(ctx, update) {
				break
			}
		}
	}
}

func (b *Bot) HandleCreateOrder(ctx context.Context, cb telego.CallbackQuery) {
	user, err := b.store.UserGet(ctx, cb.From.ID)
	if err != nil {
		b.logger.Errorf("store.UserGet: %v", err)
		return
	}

	_, err = b.store.OrderCreate(ctx, user.ID, user.TelegramID)
	if err != nil {
		b.logger.Errorf("store.OrderCreate: %s", err)
		return
	}

	_, err = b.customerBot.SendMessage(&telego.SendMessageParams{
		ChatID: telego.ChatID{ID: user.TelegramID},
		Text:   "Укажите точку подачи, водитель приедет по указанному адресу.",
	})
	if err != nil {
		b.logger.Errorf("store.SendMessage: %s", err)
	}
}

func (b *Bot) HandleArrived(ctx context.Context, cb telego.CallbackQuery) {
	orderID, err := strconv.Atoi(strings.Split(cb.Data, ":")[1])
	if err != nil {
		b.logger.Errorf("strconv.Atoi: %s", err)
		return
	}
	order, err := b.store.OrderGetByID(ctx, orderID)
	if err != nil {
		b.logger.Errorf("store.OrderGetByID(%d): %s", orderID, err)
		return
	}

	if order.TelegramID != 0 {
		_, err = b.customerBot.SendMessage(&telego.SendMessageParams{
			ChatID: telego.ChatID{ID: order.TelegramID},
			Text:   fmt.Sprintf("Водитель Вас ожидает по адресу: %s", *order.Destination),
		})
		if err != nil {
			b.logger.Errorf("store.SendMessage: %s", err)
			return
		}
	}
	_, err = b.driverBot.SendMessage(&telego.SendMessageParams{
		ChatID: telego.ChatID{ID: cb.From.ID},
		Text:   "Заказ в процессе",
		ReplyMarkup: &telego.InlineKeyboardMarkup{
			InlineKeyboard: [][]telego.InlineKeyboardButton{
				{
					{
						Text:         "Завершить заказ",
						CallbackData: fmt.Sprintf("%s:%d", finishedCallback, order.ID),
					},
				},
			},
		},
	})
	if err != nil {
		b.logger.Errorf("store.SendMessage: %s", err)
	}
}

func (b *Bot) HandleFinished(ctx context.Context, cb telego.CallbackQuery) {
	orderID, err := strconv.Atoi(strings.Split(cb.Data, ":")[1])
	if err != nil {
		b.logger.Errorf("strconv.Atoi: %s", err)
		return
	}
	order, err := b.store.OrderGetByID(ctx, orderID)
	if err != nil {
		b.logger.Errorf("store.OrderGetByID(%d): %s", orderID, err)
		return
	}

	if order.TelegramID != 0 {
		_, err = b.customerBot.SendMessage(&telego.SendMessageParams{
			ChatID: telego.ChatID{ID: order.TelegramID},
			Text:   "Ваша поездка завершена. Спасибо что воспользовались услугами нашей компании!",
		})
		if err != nil {
			b.logger.Errorf("store.SendMessage: %s", err)
			return
		}
	}
	_, err = b.driverBot.SendMessage(&telego.SendMessageParams{
		ChatID: telego.ChatID{ID: cb.From.ID},
		Text:   "Заказ завершен. Оплати пожалуйста его - https://yoomoney.ru/bill/pay/2zks2AQkgsA.230415",
	})
	if err != nil {
		b.logger.Errorf("store.SendMessage: %s", err)
	}
}

func (b *Bot) HandleGetOrder(ctx context.Context, cb telego.CallbackQuery) {
	orderID, err := strconv.Atoi(strings.Split(cb.Data, ":")[1])
	if err != nil {
		b.logger.Errorf("strconv.Atoi: %s", err)
		return
	}
	order, err := b.store.OrderGetByID(ctx, orderID)
	if err != nil {
		b.logger.Errorf("store.OrderGetByID(%d): %s", orderID, err)
		return
	}
	_, err = b.driverBot.EditMessageText(&telego.EditMessageTextParams{
		MessageID: cb.Message.MessageID,
		ChatID:    telego.ChatID{ID: b.driversChatID},
		Text:      cb.Message.Text + fmt.Sprintf("\nУже взят (%d)", cb.From.ID),
	})
	if err != nil {
		b.logger.Errorf("store.EditMessageText: %s", err)
		return
	}
	if order.TelegramID != 0 {
		_, err = b.customerBot.SendMessage(&telego.SendMessageParams{
			ChatID: telego.ChatID{ID: order.TelegramID},
			Text:   "Водитель найден. Скоро он с Вами свяжется.",
		})
		if err != nil {
			b.logger.Errorf("store.SendMessage: %s", err)
			return
		}
	}
	m := &telego.SendMessageParams{
		ChatID: telego.ChatID{ID: cb.From.ID},
		Text:   order.ToPrivate(),
		ReplyMarkup: &telego.InlineKeyboardMarkup{
			InlineKeyboard: [][]telego.InlineKeyboardButton{
				{
					{
						Text:         "Я на месте",
						CallbackData: fmt.Sprintf("%s:%d", arrivedCallback, order.ID),
					},
				},
			},
		},
	}
	_, err = b.driverBot.SendMessage(m)
	if err != nil {
		b.logger.Errorf("store.SendMessage: %s", err)
	}
}

func (b *Bot) HandleStartCommand(ctx context.Context, update telego.Update) bool {
	user, err := b.store.UserGet(ctx, update.Message.From.ID)
	if err != nil {
		b.logger.Errorf("store.UserGet: %v", err)
		return true
	}
	_, err = b.customerBot.SendMessage(&telego.SendMessageParams{
		ChatID: telego.ChatID{ID: user.TelegramID},
		Text:   "Здравствуйте, мы рады принять ваш заказ!",
		ReplyMarkup: &telego.InlineKeyboardMarkup{
			InlineKeyboard: [][]telego.InlineKeyboardButton{
				{
					{Text: "Создать заказ", CallbackData: createOrderCallback},
				},
			},
		},
	})
	if err != nil {
		b.logger.Errorf("store.SendMessage: %s", err)
	}
	return true
}

func (b *Bot) HandleMessage(ctx context.Context, update telego.Update) bool {
	message := update.Message

	if message != nil {
		command, _ := telegoutil.ParseCommand(message.Text)
		handler, ok := b.commandHandlers[command]
		if ok {
			return handler(ctx, update)
		}
	}

	cb := update.CallbackQuery
	if cb != nil {
		b.callbackHandlers[strings.Split(cb.Data, ":")[0]](ctx, *cb)
		return true
	}

	user, err := b.store.UserGet(ctx, update.Message.From.ID)
	if err != nil {
		b.logger.Errorf("store.UserGet: %v", err)
		return true
	}

	order, err := b.store.OrderGet(ctx, user.ID)
	switch {
	case err == nil:
	case errors.Is(err, storage.ErrNotFound):
		_, err = b.customerBot.SendMessage(&telego.SendMessageParams{
			ChatID: telego.ChatID{ID: user.TelegramID},
			Text:   "У тебя нет заказа",
		})
		if err != nil {
			b.logger.Errorf("customerBot.SendMessage: %v", err)
			return true
		}
	default:
		b.logger.Errorf("store.OrderGet: %v", err)
		return true
	}
	text := update.Message.Text

	if order.Source == nil {
		_, err = b.store.OrderSetField(ctx, order.ID, "source", text)
		if err != nil {
			b.logger.Errorf("store.OrderSetField: %v", err)
			return true
		}
		_, err = b.customerBot.SendMessage(&telego.SendMessageParams{
			ChatID: telego.ChatID{ID: user.TelegramID},
			Text:   "Уточните, во сколько к вам приехать?",
		})
		if err != nil {
			b.logger.Errorf("customerBot.SendMessage: %v", err)
		}
		return true
	}

	if order.Time == nil {
		_, err = b.store.OrderSetField(ctx, order.ID, "time", text)
		if err != nil {
			b.logger.Errorf("store.OrderSetField: %v", err)
			return true
		}
		_, err = b.customerBot.SendMessage(&telego.SendMessageParams{
			ChatID: telego.ChatID{ID: user.TelegramID},
			Text:   "Укажи конечную точку подачи",
		})
		if err != nil {
			b.logger.Errorf("customerBot.SendMessage: %v", err)
		}
		return true
	}

	if order.Destination == nil {
		_, err = b.store.OrderSetField(ctx, order.ID, "destination", text)
		if err != nil {
			b.logger.Errorf("store.OrderSetField: %v", err)
			return true
		}
		_, err = b.customerBot.SendMessage(&telego.SendMessageParams{
			ChatID: telego.ChatID{ID: user.TelegramID},
			Text:   "Укажите Ваш номер телефона для связи?",
			ReplyMarkup: &telego.ReplyKeyboardMarkup{
				Keyboard: [][]telego.KeyboardButton{
					{
						telego.KeyboardButton{
							Text:           "",
							RequestContact: true,
						},
					},
				},
			},
		})
		if err != nil {
			b.logger.Errorf("customerBot.SendMessage: %v", err)
		}
		return true
	}
	if order.Phone == nil {
		order, err := b.store.OrderSetField(ctx, order.ID, "phone", text)
		if err != nil {
			b.logger.Errorf("store.OrderSetField: %v", err)
			return true
		}

		_, err = b.customerBot.SendMessage(&telego.SendMessageParams{
			ChatID: telego.ChatID{ID: user.TelegramID},
			Text:   "Заказ в обработке, мы скоро найдём Вам водителя...",
		})
		if err != nil {
			b.logger.Errorf("customerBot.SendMessage: %v", err)
			return true
		}
		_, err = b.driverBot.SendMessage(&telego.SendMessageParams{
			ChatID: telego.ChatID{ID: b.driversChatID},
			Text:   order.ToDriverChat(),
			ReplyMarkup: &telego.InlineKeyboardMarkup{
				InlineKeyboard: [][]telego.InlineKeyboardButton{
					{
						{
							Text:         "Возьму заказ",
							CallbackData: fmt.Sprintf("%s:%d", orderCallback, order.ID),
						},
					},
				},
			},
		})
		if err != nil {
			b.logger.Errorf("customerBot.SendMessage: %v", err)
		}
		return true
	}

	return true
}

func (b *Bot) CheckPayments(ctx context.Context, paymentsToCheck <-chan models.Payment) {
	for payment := range paymentsToCheck {
		b.CheckPayment(ctx, payment)
	}
}

func (b *Bot) CheckPayment(ctx context.Context, payment models.Payment) {
	logger := b.logger.With(log.PaymentID(payment.ID))
	p, err := b.processor.CheckOrder(ctx, payment)
	if err != nil {
		logger.Errorf("processor.CheckOrder: %v", err)
		return
	}
	if p.Status == models.PaymentStatusPending {
		return
	}

	if p.Status != models.PaymentStatusSucceeded {
		return
	}

	order, err := b.store.OrderGetByID(ctx, payment.OrderID)
	if err != nil {
		logger.Errorf("store.OrderGetByID: %v", err)
		return
	}
	_, err = b.driverBot.SendMessage(&telego.SendMessageParams{
		ChatID:    telego.ChatID{ID: order.TelegramID},
		Text:      fmt.Sprintf(`Заказ MOSCOW-%04d успешно оплачен`, order.ID),
		ParseMode: "HTML",
	})
	if err != nil {
		logger.Errorf("sender.SendMessage: %v", err)
	}
}
