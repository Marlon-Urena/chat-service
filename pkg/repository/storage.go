package repository

import (
	"chatService/pkg/api"
	"cloud.google.com/go/firestore"
	"context"
	"encoding/json"
	jsonPatch "github.com/evanphx/json-patch/v5"
	"github.com/georgysavva/scany/pgxscan"
	"github.com/jackc/pgx/v4/pgxpool"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"log"
	"sort"
	"strconv"
)

type Storage interface {
	GetUserByIds(userIds []string) ([]*api.UserModel, error)
	GetUsersByUsernameContaining(query string) ([]*api.UserModel, error)
	UpdateConversation()
	UpdateUserConversation(patchJson []byte, uid string, conversationId string) error
	GetConversation(userId string, conversationId string) (api.Conversation, error)
	GetConversations(userId string) ([]api.Conversation, error)
	CreateConversation(newConversation api.NewConversation, userId string) (api.Conversation, error)
	AddMessage(incomingEvent api.IncomingEvent) (api.OutgoingEvent, error)
	AddParticipant(incomingEvent api.IncomingEvent) (api.OutgoingEvent, error)
}

type storage struct {
	db     *pgxpool.Pool
	client *firestore.Client
}

func (s *storage) AddMessage(incomingEvent api.IncomingEvent) (api.OutgoingEvent, error) {
	ctx := context.Background()
	var outgoingEvent api.OutgoingEvent

	messageData := incomingEvent.Message
	conversationRef := s.client.Collection("conversations").Doc(incomingEvent.ConversationId)

	// Add message to conversation collection
	messageRef, wr, err := conversationRef.Collection("messages").Add(ctx, map[string]interface{}{
		"senderId":    messageData.SenderId,
		"body":        messageData.Body,
		"contentType": messageData.ContentType,
		"createdAt":   firestore.ServerTimestamp,
	})
	if err != nil {
		log.Printf("Unable to add new document: %v", err)
		return outgoingEvent, err
	}

	// Convert document to conversation struct
	conversationSnap, _ := conversationRef.Get(ctx)
	var conversation api.ConversationDoc
	if err = conversationSnap.DataTo(&conversation); err != nil {
		log.Printf("Converting conversation snap to model struct: %v", err)
		return outgoingEvent, err
	}

	var userDocs []*firestore.DocumentRef
	for _, id := range conversation.Participants {
		userDocs = append(userDocs, s.client.Doc("users/"+id))
	}
	userSnaps, err := s.client.GetAll(context.Background(), userDocs)
	if err != nil {
		log.Println(err)
		return outgoingEvent, err
	}

	// Update each participant's user conversation document
	for _, user := range userSnaps {
		var unreadCount int
		if user.Ref.ID != messageData.SenderId {
			unreadCount = 1
		}

		// Update user conversation document
		userConversationDoc := user.Ref.Collection("conversations").Doc(conversationRef.ID)
		_, err = userConversationDoc.Update(ctx, []firestore.Update{
			{
				Path:  "unreadCount",
				Value: firestore.Increment(unreadCount),
			},
			{
				Path:  "lastUpdated",
				Value: wr.UpdateTime,
			},
		})
		if err != nil {
			log.Printf("Unable to update conversation in user collection: %s", err)
			return outgoingEvent, err
		}
	}

	outgoingEvent = api.OutgoingEvent{
		Message: &api.Message{
			Id:          (*messageRef).ID,
			Body:        messageData.Body,
			SenderId:    messageData.SenderId,
			ContentType: messageData.ContentType,
			CreatedAt:   wr.UpdateTime,
		},
		ConversationId: incomingEvent.ConversationId,
		RequestType:    incomingEvent.RequestType,
		Participants:   conversation.Participants,
	}
	log.Printf("Created message document with reference #: %s\n", (*messageRef).ID)

	return outgoingEvent, nil
}

func (s *storage) AddParticipant(incomingEvent api.IncomingEvent) (api.OutgoingEvent, error) {
	ctx := context.Background()
	var outgoingEvent api.OutgoingEvent

	newParticipants := incomingEvent.Participants
	conversationRef := s.client.Collection("conversations").Doc(incomingEvent.ConversationId)

	conversationSnap, err := conversationRef.Get(ctx)
	if status.Code(err) == codes.NotFound {
		log.Printf("Unable to find conversationRef with id %s", incomingEvent.ConversationId)
		return outgoingEvent, err
	}

	var conversation api.ConversationDoc
	if err = conversationSnap.DataTo(&conversation); err != nil {
		log.Println(err)
		return outgoingEvent, err
	}

	// Combine participants and remove duplicates
	updatedParticipants := append(conversation.Participants, newParticipants...)
	sort.Strings(updatedParticipants)
	j := 0
	for i := 1; i < len(updatedParticipants); i++ {
		if updatedParticipants[j] == updatedParticipants[i] {
			continue
		}
		j++
		updatedParticipants[j] = updatedParticipants[i]
	}
	updatedParticipants = updatedParticipants[:j+1]

	_, err = conversationRef.Update(ctx, []firestore.Update{
		{
			Path:  "participants",
			Value: updatedParticipants,
		},
	})
	if err != nil {
		log.Printf("Unable to update participants: %v", err)
		return outgoingEvent, err
	}

	outgoingEvent = api.OutgoingEvent{
		ConversationId: incomingEvent.ConversationId,
		RequestType:    incomingEvent.RequestType,
		Participants:   conversation.Participants,
	}

	log.Printf("Added new participants to conversation: %s\n", incomingEvent.ConversationId)

	return outgoingEvent, nil
}

func (s *storage) UpdateConversation() {
	//TODO implement me
	panic("implement me")
}

func (s *storage) UpdateUserConversation(patchJSON []byte, uid string, conversationId string) error {
	ctx := context.Background()

	patch, err := jsonPatch.DecodePatch(patchJSON)
	if err != nil {
		log.Fatalf("Decoding json patch: %v", err)
	}

	// Get document from user conversation collection
	userConversationDoc, err := s.client.Collection("users").Doc(uid).Collection("conversations").Doc(conversationId).Get(ctx)
	if err != nil {
		return err
	}

	// Populate struct with data from User Conversation doc
	var userConversation api.UserConversation
	if err := userConversationDoc.DataTo(&userConversation); err != nil {
		return err
	}

	// Convert User Conversation struct to binary array to be used by json patch function
	userConversationBinary, err := json.Marshal(userConversation)
	if err != nil {
		log.Fatalf("Marshalling user conversation: %v", err)
	}

	// Modify user conversation based on the instructions given from the json patch
	userConversationBinary, err = patch.Apply(userConversationBinary)
	if err != nil {
		log.Fatalf("Applying json patch to user conversation: %v\n", err)
		return err
	}

	err = json.Unmarshal(userConversationBinary, &userConversation)
	if err != nil {
		log.Fatalf("Unmarshal updated user conversation in binary: %v", err)
		return err
	}

	_, err = userConversationDoc.Ref.Set(ctx, userConversation)
	if err != nil {
		log.Printf("Setting modified data to user conversation: %v\n", err)
		return err
	}

	return nil
}

func (s *storage) GetConversation(userId string, conversationId string) (api.Conversation, error) {
	var conversation api.Conversation

	ctx := context.Background()

	// Get user conversation document snapshot
	userConversationSnap, err := s.client.Collection("users").Doc(userId).Collection("conversations").Doc(conversationId).Get(ctx)
	if err != nil {
		return conversation, err
	}

	// Convert user conversation document to struct
	var userConversation api.UserConversation
	if err := userConversationSnap.DataTo(&userConversation); err != nil {
		log.Println(err)
	}

	// Get Conversation collection from reference field value containing all conversation fields
	conversationSnap, err := userConversation.ConversationRef.Get(ctx)
	if err != nil {
		return conversation, err
	}

	// Convert conversation document to struct
	var conversationDoc api.ConversationDoc
	if err := conversationSnap.DataTo(&conversationDoc); err != nil {
		log.Println(err)
	}

	// Get the most recent messages from conversation
	query := userConversation.ConversationRef.Collection("messages").OrderBy("createdAt", firestore.Desc).Limit(20)
	messageDocs, err := query.Documents(ctx).GetAll()
	if err != nil {
		return conversation, err
	}
	var messages []api.Message
	for _, messageDoc := range messageDocs {
		var message api.Message
		if err := messageDoc.DataTo(&message); err != nil {
			log.Println(err)
		}
		message.Id = messageDoc.Ref.ID
		messages = append([]api.Message{message}, messages...)
	}

	// Get user details from db
	users, err := s.GetUserByIds(conversationDoc.Participants)
	if err != nil {
		log.Println(err)
		return conversation, err
	}

	var usersDTO []api.User
	for _, user := range users {
		userDTO := user.ConvertToDTO()
		usersDTO = append(usersDTO, userDTO)
	}

	// Construct conversation output struct
	conversation = api.Conversation{
		Id:           conversationId,
		Participants: usersDTO,
		Type:         conversation.Type,
		Messages:     messages,
		UnreadCount:  userConversation.UnreadCount,
	}

	return conversation, nil
}

func (s *storage) GetConversations(userId string) ([]api.Conversation, error) {
	ctx := context.Background()

	// Get conversations sub-collection in user collection
	path := "users/" + userId + "/conversations"
	userConversationsQuery := s.client.Collection(path).OrderBy("lastUpdated", firestore.Desc)
	userConversationSnaps, err := userConversationsQuery.Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}

	var conversations []api.Conversation
	for _, userConversationSnap := range userConversationSnaps {
		// Convert document to user conversation object
		var userConversation api.UserConversation
		if err := userConversationSnap.DataTo(&userConversation); err != nil {
			return nil, err
		}

		// Get Conversation collection from reference field value containing all conversation details
		conversationSnap, err := userConversation.ConversationRef.Get(ctx)
		if err != nil {
			return nil, err
		}

		// Convert conversation document to conversation object
		var conversation api.ConversationDoc
		if err := conversationSnap.DataTo(&conversation); err != nil {
			return nil, err
		}

		// Query for latest message documents in conversation
		query := conversationSnap.Ref.Collection("messages").OrderBy("createdAt", firestore.Desc).Limit(20)
		messageDocs, err := query.Documents(ctx).GetAll()
		if err != nil {
			return nil, err
		}

		// Converts messages to its output format
		var messages []api.Message
		for _, messageDoc := range messageDocs {
			var message api.Message
			if err := messageDoc.DataTo(&message); err != nil {
				return nil, err
			}
			message.Id = messageDoc.Ref.ID
			messages = append([]api.Message{message}, messages...)
		}

		// Get all user entities from db
		users, err := s.GetUserByIds(conversation.Participants)
		if err != nil {
			log.Println(err)
		}

		// Converts users to its output format
		var usersDTO []api.User
		for _, user := range users {
			userDTO := user.ConvertToDTO()
			usersDTO = append(usersDTO, userDTO)
		}

		// Create conversation output format
		conversationDTO := api.Conversation{
			Id:           conversationSnap.Ref.ID,
			Participants: usersDTO,
			Type:         conversation.Type,
			Messages:     messages,
		}

		conversations = append(conversations, conversationDTO)
	}

	return conversations, nil
}

func (s *storage) CreateConversation(newConversation api.NewConversation, userId string) (api.Conversation, error) {
	var conversation api.Conversation
	ctx := context.Background()

	// Retrieves users found in participants array from database
	users, err := s.GetUserByIds(newConversation.Participants)
	if err != nil {
		log.Printf("Retrieving users %v from database\n", err)
		return conversation, err
	}

	// Check if all the users exist in the database
	if users == nil || len(users) != len(newConversation.Participants) {
		log.Println("One of the users was not found")
		return conversation, err
	}

	var usersDTO []api.User
	for _, user := range users {
		userDTO := user.ConvertToDTO()
		usersDTO = append(usersDTO, userDTO)
	}

	// Query for conversations that contain exact participants
	conversations := s.client.Collection("conversations")
	conversationQuery := conversations.Where("participants", "in", newConversation.Participants)
	conversationSnaps, _ := conversationQuery.Documents(ctx).GetAll()

	// TODO perhaps I should remove this to allow multiple groups with same participants w/ maybe a group name
	// Check if a conversation already exists with the requested participants
	if len(conversationSnaps) != 0 {
		// http.Error(w, "Conversation with these participants already exist", http.StatusConflict)
		return conversation, err
	}

	conversationType := "ONE_TO_ONE"
	if len(newConversation.Participants) > 2 {
		conversationType = "GROUP"
	}

	// Create new conversation document in conversations collection
	conversationRef, _, err := s.client.Collection("conversations").Add(ctx, map[string]interface{}{
		"participants": newConversation.Participants,
		"type":         conversationType,
	})
	if err != nil {
		// http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Fatalf("Unable to add conversation to firestore: %v", err)
	}
	log.Printf("Created conversation with id: %s\n", (*conversationRef).ID)

	// Create new message document
	messageRef, _, err := conversationRef.Collection("messages").Add(ctx, map[string]interface{}{
		"senderId":    newConversation.Message.SenderId,
		"body":        newConversation.Message.Body,
		"contentType": newConversation.Message.ContentType,
		"createdAt":   firestore.ServerTimestamp,
	})
	if err != nil {
		log.Printf("Unable to add message document: %v", err)
		//http.Error(w, err.Error(), http.StatusInternalServerError)
		return conversation, nil
	}
	log.Printf("Created message document with reference #: %s\n", (*messageRef).ID)

	// Used to obtain the update timestamp
	conversationDoc, err := conversationRef.Get(ctx)
	if err != nil {
		log.Fatalf("Could not retrieve conversation document: %s", err)
	}

	var userDocs []*firestore.DocumentRef
	for _, id := range newConversation.Participants {
		userDocs = append(userDocs, s.client.Doc("users/"+id))
	}

	// Get snapshot of each participant
	userSnaps, err := s.client.GetAll(ctx, userDocs)
	if err != nil {
		log.Fatalln(err)
	}

	// Create a user conversation doc for each participant
	for _, userSnap := range userSnaps {
		var unreadCount int
		// Check if uid is the same as the conversation creator's uid
		if userSnap.Ref.ID != userId {
			unreadCount = 1
		}

		// Create new doc for user conversation
		userConversationDoc := userSnap.Ref.Collection("conversations").Doc(conversationRef.ID)
		_, err = userConversationDoc.Set(ctx, map[string]interface{}{
			"conversationRef": conversationRef,
			"unreadCount":     unreadCount,
			"updateTimestamp": conversationDoc.UpdateTime,
		})
		if err != nil {
			// http.Error(w, err.Error(), http.StatusInternalServerError)
			log.Fatalf("Unable to add conversation to user in firestore: %s", err)
		}
	}
	// Construct conversation output
	conversation = api.Conversation{
		Id:           conversationRef.ID,
		Participants: usersDTO,
		Type:         conversationType,
		Messages:     []api.Message{},
		UnreadCount:  0,
	}

	return conversation, nil
}

func (s *storage) GetUserByIds(uIds []string) ([]*api.UserModel, error) {
	var users []*api.UserModel
	ids := make([]interface{}, len(uIds))
	ids[0] = uIds[0]
	inStmt := "$1"
	for i := 1; i < len(uIds); i++ {
		inStmt = inStmt + ",$" + strconv.Itoa(i+1)
		ids[i] = uIds[i]
	}
	if err := pgxscan.Select(context.Background(), s.db, &users, "SELECT * FROM user_account WHERE uid IN ("+inStmt+")", ids...); err != nil {
		return nil, err
	}
	return users, nil
}

func (s *storage) GetUsersByUsernameContaining(query string) ([]*api.UserModel, error) {
	var users []*api.UserModel
	if err := pgxscan.Select(context.Background(), s.db, &users, "SELECT * FROM user_account WHERE username LIKE %$1%", query); err != nil {
		return nil, err
	}
	return users, nil
}

func NewStorage(db *pgxpool.Pool, client *firestore.Client) Storage {
	return &storage{db: db, client: client}
}
