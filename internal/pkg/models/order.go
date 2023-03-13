package models

import (
	"fmt"

	"github.com/google/uuid"
)

type Order struct {
	ID          int
	UserID      *uuid.UUID
	TelegramID  int64
	Source      *string
	Destination *string
	Phone       *string
	Time        *string
}

func (o *Order) ToDriverChat() string {
	return fmt.Sprintf(`ID: MOSCOW-%04d
Откуда: %s
Куда: %s
Время: %s`, o.ID, *o.Source, *o.Destination, *o.Time)
}

func (o *Order) ToPrivate() string {
	return fmt.Sprintf(`ID: MOSCOW-%04d
Откуда: %s
Куда: %s
Время: %s
Телефон: %s`, o.ID, *o.Source, *o.Destination, *o.Time, *o.Phone)
}
