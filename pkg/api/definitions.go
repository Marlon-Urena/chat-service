package api

import (
	"cloud.google.com/go/firestore"
	"time"
)

type ConversationDoc struct {
	Participants []string `firestore:"participants"`
	Type         string   `firestore:"type"`
}

type NewConversation struct {
	Participants []string `json:"participants"`
	Message      Message  `json:"message"`
}

type Conversation struct {
	Id           string    `json:"id"`
	Participants []User    `json:"participants"`
	Type         string    `json:"type"`
	Messages     []Message `json:"messages"`
	UnreadCount  int       `json:"unreadCount"`
}

type UserConversation struct {
	UnreadCount     int                    `firestore:"unreadCount" json:"unreadCount"`
	ConversationRef *firestore.DocumentRef `firestore:"conversationRef" json:"conversationRef"`
}

type Message struct {
	Id          string    `firestore:"id,omitempty" json:"id,omitempty"`
	SenderId    string    `firestore:"senderId" json:"senderId"`
	ContentType string    `firestore:"contentType" json:"contentType"`
	Body        string    `firestore:"body" json:"body,omitempty"`
	CreatedAt   time.Time `firestore:"createdAt" json:"createdAt,omitempty"`
	Attachments []string  `firestore:"attachments,omitempty" json:"attachments,omitempty"`
}

type User struct {
	Id           string    `json:"id"`
	Email        string    `json:"email"`
	Username     string    `json:"username"`
	Name         *string   `json:"name"`
	Avatar       *string   `json:"avatar"`
	Status       string    `json:"status"`
	Position     *string   `json:"position"`
	PhoneNumber  *string   `json:"phoneNumber"`
	Address      *string   `json:"address"`
	LastActivity time.Time `json:"lastActivity"`
}

type IncomingEvent struct {
	ConversationId string   `json:"conversationId,omitempty"`
	RequestType    int      `json:"requestType,omitempty"`
	Message        *Message `json:"message,omitempty"`
	Participants   []string `json:"participants,omitempty"`
	Token          string   `json:"token,omitempty"`
}

type OutgoingEvent struct {
	ConversationId string   `json:"conversationId,omitempty"`
	RequestType    int      `json:"requestType,omitempty"`
	Message        *Message `json:"message,omitempty"`
	Participants   []string `json:"participants,omitempty"`
	Client         *Client
}

type UserModel struct {
	UID          string
	FirstName    *string
	LastName     *string
	Username     string
	Email        string
	Address      *string
	City         *string
	State        *string
	Country      *string
	ZipCode      *string
	PhotoUrl     *string
	PhoneNumber  *string
	Status       string
	LastActivity time.Time
}

func (u *UserModel) ConvertToDTO() User {
	var name string
	if u.FirstName != nil && u.LastName != nil {
		name = *u.FirstName + " " + *u.LastName
	}
	return User{
		Id:           u.UID,
		Email:        u.Email,
		Username:     u.Username,
		Name:         &name,
		Avatar:       u.PhotoUrl,
		Status:       u.Status,
		Position:     nil,
		PhoneNumber:  u.PhoneNumber,
		Address:      u.Address,
		LastActivity: u.LastActivity,
	}
}
