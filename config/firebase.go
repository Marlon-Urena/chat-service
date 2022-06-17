package config

import (
	"context"
	firebase "firebase.google.com/go/v4"
	"log"
	"sync"
)

var firebaseOnce sync.Once
var firebaseApp *firebase.App

func SetupFirebase() *firebase.App {
	firebaseOnce.Do(func() {
		app, err := firebase.NewApp(context.Background(), nil)
		if err != nil {
			log.Fatalln(err)
		}
		firebaseApp = app
	})

	return firebaseApp
}
