package api

type ChatService interface {
	AddMessage(incomingEvent IncomingEvent) (OutgoingEvent, error)
	AddParticipant(incomingEvent IncomingEvent) (OutgoingEvent, error)
	UpdateConversation()
	UpdateUserConversation(patchJson []byte, userId string, conversationId string) error
	GetConversation(userId string, conversationId string) (Conversation, error)
	GetConversations(userId string) ([]Conversation, error)
	CreateConversation(newConversation NewConversation, userId string) (Conversation, error)
}

type ChatRepository interface {
	AddMessage(incomingEvent IncomingEvent) (OutgoingEvent, error)
	AddParticipant(incomingEvent IncomingEvent) (OutgoingEvent, error)
	UpdateConversation()
	UpdateUserConversation(patchJson []byte, userId string, conversationId string) error
	GetConversation(userId string, conversationId string) (Conversation, error)
	GetConversations(userId string) ([]Conversation, error)
	CreateConversation(newConversation NewConversation, userId string) (Conversation, error)
}

type chatService struct {
	storage ChatRepository
}

func NewChatService(storage ChatRepository) ChatService {
	return &chatService{storage: storage}
}

func (c *chatService) UpdateConversation() {
	//TODO implement me
	panic("implement me")
}

func (c *chatService) UpdateUserConversation(patchJson []byte, userId string, conversationId string) error {
	err := c.storage.UpdateUserConversation(patchJson, userId, conversationId)

	if err != nil {
		return err
	}

	return nil
}

func (c *chatService) GetConversation(userId string, conversationId string) (Conversation, error) {
	conversation, err := c.storage.GetConversation(userId, conversationId)

	if err != nil {
		return conversation, err
	}

	return conversation, nil

}

func (c *chatService) GetConversations(userId string) ([]Conversation, error) {
	conversations, err := c.storage.GetConversations(userId)

	if err != nil {
		return conversations, err
	}

	return conversations, err
}

func (c *chatService) CreateConversation(newConversation NewConversation, userId string) (Conversation, error) {
	conversation, err := c.storage.CreateConversation(newConversation, userId)

	if err != nil {
		return conversation, err
	}

	return conversation, nil
}

func (c *chatService) AddMessage(incomingEvent IncomingEvent) (OutgoingEvent, error) {
	outgoingEvent, err := c.storage.AddMessage(incomingEvent)

	if err != nil {
		return outgoingEvent, err
	}

	return outgoingEvent, nil
}

func (c *chatService) AddParticipant(incomingEvent IncomingEvent) (OutgoingEvent, error) {
	outgoingEvent, err := c.storage.AddParticipant(incomingEvent)

	if err != nil {
		return outgoingEvent, err
	}

	return outgoingEvent, nil
}
