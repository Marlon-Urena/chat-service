package handler

import (
	"cloud.google.com/go/firestore"
	"encoding/json"
	"github.com/go-chi/chi/v5"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"log"
	"messageService/dto"
	"messageService/model"
	"messageService/repository"
	"net/http"
)

func GetConversation(repository repository.UserRepository) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		// Retrieve firestore client from context
		client := r.Context().Value("firestore").(*firestore.Client)
		// UID from Access Token contained in Authorization header
		uid := r.Context().Value("UID").(string)

		conversationId := chi.URLParam(r, "conversationId")

		// Get Conversation subcollection in User collection
		userConversationSnap, err := client.Collection("users").Doc(uid).Collection("conversations").Doc(conversationId).Get(r.Context())
		if status.Code(err) == codes.NotFound {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		}
		var userConversation model.UserConversationModel
		if err := userConversationSnap.DataTo(&userConversation); err != nil {
			log.Println(err)
		}

		// Get Conversation collection from reference field value containing all conversation details
		conversationSnap, err := userConversation.ConversationRef.Get(r.Context())
		if status.Code(err) == codes.NotFound {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		}
		var conversation model.ConversationModel
		if err := conversationSnap.DataTo(&conversation); err != nil {
			log.Println(err)
		}

		query := userConversation.ConversationRef.Collection("messages").OrderBy("createdAt", firestore.Desc).Limit(1)
		messageDocs, err := query.Documents(r.Context()).GetAll()
		if status.Code(err) == codes.NotFound {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		}

		var messages []dto.MessageDTO
		for _, messageDoc := range messageDocs {
			var message dto.MessageDTO
			if err := messageDoc.DataTo(&message); err != nil {
				log.Println(err)
			}
			message.Id = messageDoc.Ref.ID
			messages = append(messages, message)
		}

		users, err := repository.FindByIDs(conversation.Participants)
		if err != nil {
			log.Println(err)
		}

		var usersDTO []dto.UserDTO
		for _, user := range users {
			userDTO := user.ConvertToDTO()
			usersDTO = append(usersDTO, userDTO)
		}

		conversationDTO := dto.ConversationDTO{
			Id:           conversationId,
			Participants: usersDTO,
			Type:         conversation.Type,
			Messages:     messages,
		}
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(conversationDTO); err != nil {
			log.Printf("Unable to encode conversation data: %v\n", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		log.Printf("Successfully retrieved conversation with id: %s", conversationId)
	}
}
