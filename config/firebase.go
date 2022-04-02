package config

import (
	"context"
	firebase "firebase.google.com/go/v4"
	"log"
)

func SetupFirebase() *firebase.App {
	app, err := firebase.NewApp(context.Background(), nil)
	if err != nil {
		log.Fatalln(err)
	}
	return app
}
