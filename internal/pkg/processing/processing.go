package processing

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/andrey-berenda/perfect-driver/internal/pkg/models"
	"github.com/andrey-berenda/perfect-driver/internal/pkg/storage"
)

type Processor interface {
	CreatePayment(ctx context.Context, orderID int) (string, error)
	CheckOrder(ctx context.Context, payment models.Payment) (*models.Payment, error)
}

type youMoneyProcessor struct {
	store            *storage.Store
	httpClient       *http.Client
	authorization    string
	createOrderURL   string
	buildGetOrderURL func(orderID uuid.UUID) string
}

type Amount struct {
	Value    string `json:"value"`
	Currency string `json:"currency"`
}

type Confirmation struct {
	Type            string `json:"type"`
	ReturnURL       string `json:"return_url"`
	ConfirmationURL string `json:"confirmation_url"`
}

type Customer struct {
	Email string `json:"email"`
}

type CreateOrderRequest struct {
	Amount            Amount       `json:"amount"`
	Capture           bool         `json:"capture"`
	Confirmation      Confirmation `json:"confirmation"`
	Description       string       `json:"description"`
	SavePaymentMethod bool         `json:"save_payment_method"`
}

type PaymentMethod struct {
	ID *uuid.UUID `json:"id"`
}

type OrderResponse struct {
	ID            uuid.UUID            `json:"id"`
	Status        models.PaymentStatus `json:"status"`
	Paid          bool                 `json:"paid"`
	Amount        Amount               `json:"amount"`
	PaymentMethod *PaymentMethod       `json:"payment_method"`
	Confirmation  Confirmation         `json:"confirmation"`
	CreatedAt     time.Time            `json:"created_at"`
	Description   string               `json:"description"`
}

func newRequest() CreateOrderRequest {
	return CreateOrderRequest{
		Amount: Amount{
			Currency: "RUB",
			Value:    "500",
		},
		Capture: true,
		Confirmation: Confirmation{
			Type:      "redirect",
			ReturnURL: "https://t.me/PerfectDriverBot",
		},
		Description: "Оплата",
	}
}

func New(
	store *storage.Store,
	httpClient *http.Client,
	authorization string,
	createOrderURL string,
	buildGetOrderURL func(orderID uuid.UUID) string,
) Processor {
	return &youMoneyProcessor{
		store:            store,
		httpClient:       httpClient,
		authorization:    authorization,
		createOrderURL:   createOrderURL,
		buildGetOrderURL: buildGetOrderURL,
	}
}

func (p *youMoneyProcessor) CreatePayment(ctx context.Context, orderID int) (string, error) {
	body, err := json.MarshalIndent(newRequest(), "", "  ")
	if err != nil {
		return "", fmt.Errorf("json.Marshal: %w", err)
	}

	httpRequest, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		p.createOrderURL,
		bytes.NewReader(body),
	)
	if err != nil {
		return "", fmt.Errorf("http.NewRequestWithContext: %w", err)
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.Header.Set("Idempotence-Key", uuid.New().String())
	httpRequest.Header.Set("Authorization", fmt.Sprintf("Basic %s", p.authorization))

	response, err := p.httpClient.Do(httpRequest)
	if err != nil {
		return "", fmt.Errorf("httpClient.Do: %w", err)
	}

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return "", fmt.Errorf("io.ReadAll(response.Body): %w", err)
	}

	if err = response.Body.Close(); err != nil {
		return "", fmt.Errorf("response.Body.Close: %w", err)
	}

	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("httpClient.Do: returned status %d: %s", response.StatusCode, string(responseBody))
	}

	resp := OrderResponse{}
	if err = json.Unmarshal(responseBody, &resp); err != nil {
		return "", fmt.Errorf("json.Unmarshal(responseBody): %w", err)
	}
	payment := models.Payment{
		ID:              resp.ID,
		OrderID:         orderID,
		Status:          models.PaymentStatusPending,
		ConfirmationURL: resp.Confirmation.ConfirmationURL,
	}

	if err = p.store.PaymentCreate(ctx, payment); err != nil {
		return "", fmt.Errorf("store.PaymentCreate: %w", err)
	}

	return resp.Confirmation.ConfirmationURL, nil
}

func (p *youMoneyProcessor) CheckOrder(ctx context.Context, payment models.Payment) (*models.Payment, error) {
	httpRequest, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		p.buildGetOrderURL(payment.ID),
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("http.NewRequestWithContext: %w", err)
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.Header.Set("Authorization", fmt.Sprintf("Basic %s", p.authorization))
	response, err := p.httpClient.Do(httpRequest)
	if err != nil {
		return nil, fmt.Errorf("httpClient.Do: %w", err)
	}
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("io.ReadAll(response.Body): %w", err)
	}
	if err = response.Body.Close(); err != nil {
		return nil, fmt.Errorf("response.Body.Close: %w", err)
	}

	if response.StatusCode == http.StatusNotFound {
		err = p.store.PaymentSetStatus(ctx, payment.ID, models.PaymentStatusCanceled)
		if err != nil {
			return nil, fmt.Errorf("store.PaymentSetStatus(%s): %w", models.PaymentStatusCanceled, err)
		}
		payment.Status = models.PaymentStatusCanceled
		return &payment, nil
	}

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("httpClient.Do: returned status %d: %s", response.StatusCode, string(responseBody))
	}
	resp := OrderResponse{}
	if err = json.Unmarshal(responseBody, &resp); err != nil {
		return nil, fmt.Errorf("json.Unmarshal(responseBody): %w", err)
	}
	err = p.store.PaymentSetStatus(ctx, payment.ID, resp.Status)
	if err != nil {
		return nil, fmt.Errorf("store.PaymentSetStatus(%s): %w", resp.Status, err)
	}
	payment.Status = resp.Status
	return &payment, nil
}
