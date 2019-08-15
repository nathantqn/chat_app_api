package chat_app

import (
	"time"
)

type Message struct {
	Text      string `json:"text"`
	UserID    int
	ID        int `json:"id"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
	RoomID    int
}
