package app

import (
	"chatService/pkg/api"
	"context"
	"github.com/go-chi/chi/v5"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type Server struct {
	router      *chi.Mux
	userService api.UserService
	chatService api.ChatService
}

func NewServer(router *chi.Mux, userService api.UserService, chatService api.ChatService) *Server {
	return &Server{
		router:      router,
		userService: userService,
		chatService: chatService,
	}
}

func (s *Server) Run() error {
	hub := api.NewHub()
	go hub.Run()

	// run function that initializes the routes
	r := s.Routes(hub)

	server := &http.Server{Addr: os.Getenv("SERVER_URL"), Handler: r}

	// Server run context
	serverCtx, serverStopCtx := context.WithCancel(context.Background())

	// Listen for syscall signals for process to interrupt/quit
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		<-sig

		// Shutdown signal with grace period of 30 seconds
		shutdownCtx, cancelFunc := context.WithTimeout(serverCtx, 30*time.Second)

		// Cancels shutdownCtx if shutdown occurs before timeout
		defer cancelFunc()

		go func() {
			<-shutdownCtx.Done()
			if shutdownCtx.Err() == context.DeadlineExceeded {
				log.Fatal("graceful shutdown timed out.. forcing exit.")
			}
		}()

		// Trigger graceful shutdown
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Fatal(err)
		}
		serverStopCtx()
	}()

	if err := server.ListenAndServe(); err != nil {
		return err
	}

	return nil
}
