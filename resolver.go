package chat_app

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"time"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
) // THIS CODE IS A STARTING POINT ONLY. IT WILL NOT BE UPDATED WITH SCHEMA CHANGES.

var db *gorm.DB

func init() {
	initDB()
}

func initDB() {
	var err error
	db, err = gorm.Open("postgres", "user=nhattruong dbname=go_chat_app password=Sandworm1$ sslmode=disable")
	if err != nil {
		panic("Cannot connect to database")
	}
	// Enable Logger, show detailed log
	db.LogMode(true)
	db.AutoMigrate(&User{})
	db.AutoMigrate(&Room{})
	db.AutoMigrate(&Message{})
}

type UserChannel chan *Message

type UserChannels map[int]UserChannel
type ChatRooms map[int]UserChannels

type Resolver struct {
	mu        sync.Mutex
	ChatRooms ChatRooms
}

func (r *Resolver) Message() MessageResolver {
	return &messageResolver{r}
}
func (r *Resolver) Room() RoomResolver {
	return &roomResolver{r}
}
func (r *Resolver) User() UserResolver {
	return &userResolver{r}
}
func (r *Resolver) Mutation() MutationResolver {
	return &mutationResolver{r}
}
func (r *Resolver) Query() QueryResolver {
	return &queryResolver{r}
}
func (r *Resolver) Subscription() SubscriptionResolver {
	return &subscriptionResolver{r}
}

type userResolver struct{ *Resolver }

func getTimeNowInMilisecond() int64 {
	now := time.Now()
	unixNano := now.UnixNano()
	umillisec := unixNano / 1000000
	return umillisec
}

func (r *userResolver) Rooms(ctx context.Context, obj *User) ([]*Room, error) {
	var rooms []*Room
	db.Model(&obj).Related(&rooms, "Rooms")
	return rooms, nil
}

type roomResolver struct{ *Resolver }

func (r *roomResolver) Messages(ctx context.Context, obj *Room) ([]*Message, error) {
	var messages []*Message
	if err := db.Where("room_id = ?", obj.ID).Find(&messages).Error; gorm.IsRecordNotFoundError(err) {
		return nil, errors.New("Not Found")
	}

	return messages, nil
}

func (r *roomResolver) Users(ctx context.Context, obj *Room) ([]*User, error) {
	var users []*User
	db.Model(&obj).Related(&users, "Users")
	return users, nil
}

type messageResolver struct{ *Resolver }

func (r *messageResolver) User(ctx context.Context, obj *Message) (*User, error) {
	var user User
	if err := db.First(&user, obj.UserID).Error; gorm.IsRecordNotFoundError(err) {
		return nil, errors.New("Not Found")
	}
	return &user, nil
}

func (r *messageResolver) Room(ctx context.Context, obj *Message) (*Room, error) {
	var room Room
	if err := db.First(&room, obj.RoomID).Error; gorm.IsRecordNotFoundError(err) {
		return nil, errors.New("Not Found")
	}
	return &room, nil
}

type mutationResolver struct{ *Resolver }

func (r *mutationResolver) CreateMessage(ctx context.Context, input NewMessage) (*Message, error) {
	var message = &Message{Text: input.Text, UserID: input.UserID, RoomID: input.RoomID}
	if err := db.Create(message).Error; err != nil {
		return nil, err
	}
	r.mu.Lock()
	if len(r.ChatRooms) != 0 && len(r.ChatRooms[input.RoomID]) != 0 {
		for userChannel := range r.ChatRooms[input.RoomID] {
			r.ChatRooms[input.RoomID][userChannel] <- message
		}
	}
	r.mu.Unlock()
	return message, nil
}

func (r *mutationResolver) CreateUser(ctx context.Context, input NewUser) (*User, error) {
	randID := getTimeNowInMilisecond()
	avatarURL := "https://api.adorable.io/avatars/50/" + strconv.FormatInt(randID, 10) + input.Name + ".png"
	user := &User{Name: input.Name, AvatarURL: avatarURL}
	db.Create(user)
	return user, nil
}

func (r *mutationResolver) CreateRoom(ctx context.Context, input NewRoom) (*Room, error) {
	room := &Room{Name: input.Name}
	db.Create(room)
	r.JoinRoom(ctx, NewParticipation{UserID: input.UserID, RoomID: room.ID})
	return room, nil
}

func (r *mutationResolver) JoinRoom(ctx context.Context, input NewParticipation) (*Room, error) {
	var user User
	var room Room
	db.Find(&user, input.UserID)
	db.Find(&room, input.RoomID)
	db.Model(&user).Association("Rooms").Append(room)
	return &room, nil
}

func (r *mutationResolver) Login(ctx context.Context, input LoginInput) (*User, error) {
	var user User
	if err := db.Where("name = ?", input.Name).First(&user).Error; gorm.IsRecordNotFoundError(err) {
		newUser, errCreateNewUser := r.CreateUser(ctx, NewUser{Name: input.Name})
		return newUser, errCreateNewUser
	}
	return &user, nil
}

type queryResolver struct{ *Resolver }

func (r *queryResolver) Messages(ctx context.Context) ([]*Message, error) {
	var messages []*Message
	db.Find(&messages)
	return messages, nil
}

func (r *queryResolver) User(ctx context.Context, id int) (*User, error) {
	var user User
	if err := db.First(&user, id).Error; gorm.IsRecordNotFoundError(err) {
		return nil, errors.New("Not Found")
	}
	return &user, nil
}

func (r *queryResolver) Room(ctx context.Context, id int) (*Room, error) {
	var room Room
	if err := db.First(&room, id).Error; gorm.IsRecordNotFoundError(err) {
		return nil, errors.New("Not Found")
	}
	return &room, nil
}

func (r *queryResolver) Rooms(ctx context.Context) ([]*Room, error) {
	var rooms []*Room
	db.Find(&rooms)
	return rooms, nil
}

type subscriptionResolver struct{ *Resolver }

func (r *subscriptionResolver) MessageCreated(ctx context.Context, roomID int, userID int) (<-chan *Message, error) {
	events := make(chan *Message, 1)
	go func() {
		<-ctx.Done()
		r.mu.Lock()
		delete(r.ChatRooms[roomID], userID)
		r.mu.Unlock()
	}()

	r.mu.Lock()
	if len(r.ChatRooms) == 0 {
		r.ChatRooms = ChatRooms{roomID: UserChannels{userID: events}}
	} else if len(r.ChatRooms[roomID]) == 0 {
		r.ChatRooms[roomID] = UserChannels{userID: events}
	} else {
		r.ChatRooms[roomID][userID] = events
	}
	r.mu.Unlock()
	return events, nil
}
