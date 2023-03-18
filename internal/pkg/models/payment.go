package models

import (
	"github.com/google/uuid"
)

type PaymentStatus string

const (
	PaymentStatusPending           PaymentStatus = "pending"
	PaymentStatusSucceeded         PaymentStatus = "succeeded"
	PaymentStatusWaitingForCapture PaymentStatus = "waiting_for_capture"
	PaymentStatusCanceled          PaymentStatus = "canceled"
)

type Payment struct {
	ID              uuid.UUID
	OrderID         int
	ConfirmationURL string
	Status          PaymentStatus
}
