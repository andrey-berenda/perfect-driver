package models

import "github.com/google/uuid"

type UserStep string

type User struct {
	ID         uuid.UUID
	TelegramID int64
}
