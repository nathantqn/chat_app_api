package chat_app

import "time"

type Room struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
	Messages  []Message
	Users     []*User `gorm:"many2many:user_rooms;"`
}
