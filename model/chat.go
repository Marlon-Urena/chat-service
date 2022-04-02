package model

import (
	"cloud.google.com/go/firestore"
	"messageService/dto"
	"time"
)

type ConversationModel struct {
	Participants []string                 `firestore:"participants"`
	Type         string                   `firestore:"type"`
	Messages     *firestore.CollectionRef `firestore:"messages"`
}

type UserConversationModel struct {
	UnreadCount     int                    `firestore:"unreadCount"`
	ConversationRef *firestore.DocumentRef `firestore:"conversationRef"`
}

type MessageModel struct {
	Id          string    `firestore:"id,omitempty"`
	SenderId    string    `firestore:"senderId"`
	ContentType string    `firestore:"contentType"`
	Body        string    `firestore:"body"`
	CreatedAt   time.Time `firestore:"createdAt"`
	Attachments []string  `firestore:"attachments,omitempty"`
}

func (m MessageModel) convertToDTO() *dto.MessageDTO {
	return &dto.MessageDTO{
		Id:          m.Id,
		SenderId:    m.SenderId,
		ContentType: m.ContentType,
		Body:        m.Body,
		CreatedAt:   m.CreatedAt,
		Attachments: m.Attachments,
	}
}
