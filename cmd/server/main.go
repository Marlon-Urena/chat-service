package main

import (
	"chatService/config"
	"chatService/pkg/api"
	"chatService/pkg/app"
	"chatService/pkg/repository"
	"context"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/joho/godotenv"
	"log"
	"os"
)

func init() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatalln("Error loading .env file")
	}
}

func main() {
	db, err := setupDatabase()
	if err != nil {
		log.Printf("Unable to connect to database: %v", err)
		os.Exit(1)
	}
	log.Println("Successfully connected to database")

	defer db.Close()

	firebaseApp := config.SetupFirebase()

	firestore, err := firebaseApp.Firestore(context.Background())
	if err != nil {
		log.Fatalln(err)
	}

	storage := repository.NewStorage(db, firestore)

	router := chi.NewRouter()

	userService := api.NewUserService(storage)

	chatService := api.NewChatService(storage)

	server := app.NewServer(router, userService, chatService)

	if err = server.Run(); err != nil {
		log.Println(err)
	}
}

func setupDatabase() (*pgxpool.Pool, error) {
	conn, err := pgxpool.Connect(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		return nil, err
	}

	return conn, nil
}
