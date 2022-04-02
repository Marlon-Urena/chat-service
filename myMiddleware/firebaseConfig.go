package myMiddleware

import (
	"context"
	firebase "firebase.google.com/go/v4"
	"log"
	"net/http"
)

// FirebaseConfig /* FirebaseConfig HTTP myMiddleware setting Firebase Auth and Firestore
//on request's context
func FirebaseConfig(firebaseApp *firebase.App) func(next http.Handler) http.Handler {
	auth, err := firebaseApp.Auth(context.Background())
	if err != nil {
		log.Fatalln(err)
	}

	firestore, err := firebaseApp.Firestore(context.Background())
	if err != nil {
		log.Fatalln(err)
	}

	return func(next http.Handler) http.Handler {

		fn := func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), "auth", auth)
			ctx = context.WithValue(ctx, "firestore", firestore)

			next.ServeHTTP(w, r.WithContext(ctx))
		}
		return http.HandlerFunc(fn)
	}
}
