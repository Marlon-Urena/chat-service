package main

import (
	"context"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v4"
	"github.com/joho/godotenv"
	"log"
	"messageService/config"
	"messageService/handler"
	"messageService/myMiddleware"
	"messageService/repository"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func init() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatalln("Error loading .env file")
	}
}

func main() {
	hub := newHub()
	go hub.run()

	ctx := context.Background()
	conn := config.SetupDatabase(ctx)
	db := &config.Database{DB: conn}

	defer func(conn *pgx.Conn, ctx context.Context) {
		if err := conn.Close(ctx); err != nil {
			log.Fatal(err)
		}
		log.Println("Database has been successfully closed")
	}(db.DB, ctx)

	userRepository := repository.UserRepository{DB: db.DB}

	r := chi.NewRouter()
	r.Use(cors.Handler(cors.Options{
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: false,
		MaxAge:           300, // Maximum value not ignored by any of major browsers
	}))
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(myMiddleware.FirebaseConfig(config.SetupFirebase()))
	r.Use(myMiddleware.Authenticator)

	r.Get("/chat", func(w http.ResponseWriter, r *http.Request) {
		serveWs(hub, w, r)
	})
	r.Get("/chat/conversation/{conversationId}", handler.GetConversation(userRepository))

	server := &http.Server{Addr: "localhost:3003", Handler: r}

	// Server run context
	serverCtx, serverStopCtx := context.WithCancel(ctx)

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
		log.Fatal("ListenAndServe: ", err)
	}
}
