package chat_app

import "time"

type User struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatarURL"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
	Rooms     []*Room `gorm:"many2many:user_rooms;"`
}
