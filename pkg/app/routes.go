package app

import (
	"chatService/config"
	"chatService/pkg/api"
	myMiddleware "chatService/pkg/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func (s *Server) Routes(hub *api.Hub) *chi.Mux {
	r := s.router
	r.Use(cors.Handler(cors.Options{
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: false,
		MaxAge:           300, // Maximum value not ignored by any of major browsers
	}))
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(myMiddleware.FirebaseConfig(config.SetupFirebase()))

	r.Route("/chat", func(r chi.Router) {
		r.Use(myMiddleware.Authenticator)
		r.Get("/conversation/{conversationId}", s.GetConversation())
		r.Post("/conversation", s.CreateConversation())
		r.Get("/conversation", s.GetConversations())
		r.Patch("/conversation/{conversationId}", s.UpdateConversation())
		r.Patch("/user/conversation/{conversationId}", s.UpdateUserConversation())
	})

	r.Get("/chat/ws", s.ServeWs(hub))

	return r
}
