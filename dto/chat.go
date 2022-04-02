package dto

import "time"

type ConversationDTO struct {
	Id           string       `json:"id,omitempty"`
	Participants []UserDTO    `json:"participants"`
	Type         string       `json:"type"`
	Messages     []MessageDTO `json:"messages"`
}

type MessageDTO struct {
	Id          string    `json:"id,omitempty"`
	SenderId    string    `json:"senderId"`
	ContentType string    `json:"contentType"`
	Body        string    `json:"body"`
	CreatedAt   time.Time `json:"createdAt"`
	Attachments []string  `json:"attachments,omitempty"`
}

type UserDTO struct {
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
