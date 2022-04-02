package myMiddleware

import (
	"context"
	"firebase.google.com/go/v4/auth"
	"net/http"
	"strings"
)

func Authenticator(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		firebaseAuth := r.Context().Value("auth").(*auth.Client)

		authorizationToken := r.Header.Get("Authorization")

		idToken := strings.TrimSpace(strings.Replace(authorizationToken,
			"Bearer", "", 1))

		token, err := firebaseAuth.VerifyIDToken(context.Background(), idToken)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), "UID", token.UID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
