package app

import (
	"chatService/pkg/api"
	"cloud.google.com/go/firestore"
	"encoding/json"
	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"io/ioutil"
	"log"
	"net/http"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  8092,
	WriteBufferSize: 8092,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func (s *Server) UpdateConversation() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Method not implemented", http.StatusNotImplemented)
	}
}

func (s *Server) UpdateUserConversation() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// UID from Access Token contained in Authorization header
		uid := r.Context().Value("UID").(string)

		conversationId := chi.URLParam(r, "conversationId")

		patchJSON, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Fatalf("Reading bytes from request: %v", err)
		}

		if err := s.chatService.UpdateUserConversation(patchJSON, uid, conversationId); err != nil {
			http.Error(w, "Couldn't process request", http.StatusBadRequest)
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func (s *Server) GetConversation() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// UID from Access Token contained in Authorization header
		uid := r.Context().Value("UID").(string)

		conversationId := chi.URLParam(r, "conversationId")

		conversation, err := s.chatService.GetConversation(uid, conversationId)
		if err != nil {
			http.Error(w, "Error getting conversation with conversation id:"+conversationId, http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(conversation); err != nil {
			log.Printf("Unable to encode conversation data: %v\n", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		log.Printf("Successfully retrieved conversation with id: %s", conversationId)
	}
}

func (s *Server) GetConversations() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// UID from Access Token contained in Authorization header
		uid := r.Context().Value("UID").(string)

		conversations, err := s.chatService.GetConversations(uid)

		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(conversations); err != nil {
			log.Printf("Unable to encode conversation data: %v\n", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		log.Printf("Successfully retrieved conversations for user with id: %s", uid)
	}
}

func (s *Server) GetContacts() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		query := chi.URLParam(r, "query")

		users, err := s.userService.GetUsersByUsernameContaining(query)
		if err != nil {
			log.Println(err)
		}

		var usersDTO []api.User
		for _, user := range users {
			userDTO := user.ConvertToDTO()
			usersDTO = append(usersDTO, userDTO)
		}

		w.WriteHeader(http.StatusOK)

		if err := json.NewEncoder(w).Encode(usersDTO); err != nil {
			log.Printf("Unable to encode users data: %v\n", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		log.Printf("Successfully retrieved users like: %s", query)
	}
}

func (s *Server) MarkConversationAsRead() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Retrieve firestore client from context
		client := r.Context().Value("firestore").(*firestore.Client)
		// UID from Access Token contained in Authorization header
		uid := r.Context().Value("UID").(string)

		conversationId := chi.URLParam(r, "conversationId")

		// Get Conversation sub-collection in User collection
		userRef := client.Collection("users").Doc(uid)
		conversationRef := userRef.Collection("conversations").Doc(conversationId)
		_, err := conversationRef.Update(r.Context(), []firestore.Update{
			{
				Path:  "unreadCount",
				Value: 0,
			},
		})
		if status.Code(err) == codes.NotFound {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		}

		w.WriteHeader(http.StatusOK)
		log.Printf("Marked conversation with id %s as read\n", conversationId)
	}
}

func (s *Server) CreateConversation() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// UID from Access Token contained in Authorization header
		uid := r.Context().Value("UID").(string)

		var newConversation api.NewConversation
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&newConversation); err != nil {
			log.Printf("Unable to unmarshal request body: %v\n", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		conversation, err := s.chatService.CreateConversation(newConversation, uid)

		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}

		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(conversation); err != nil {
			log.Printf("Unable to encode conversation data: %v\n", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}
}

func (s *Server) ServeWs(hub *api.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		uid := r.URL.Query().Get("uid")
		if uid == "" {
			log.Println("uid in query param required")
			return
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println(err)
			return
		}

		log.Println("Connected to websocket")
		client := api.NewClient(hub, conn, make(chan []byte, 256), uid, s.chatService)
		client.Hub.Register <- client

		// Allow collection of memory referenced by the caller by doing all work in
		// new goroutines.
		go client.WritePump()
		go client.ReadPump()
	}
}
