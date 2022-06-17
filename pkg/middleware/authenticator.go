package middleware

import (
	"context"
	"firebase.google.com/go/v4/auth"
	"net/http"
	"strings"
)

func Authenticator(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		firebaseAuth := r.Context().Value("auth").(*auth.Client)

		idToken := findToken(r, tokenFromHeader, tokenFromQuery)

		token, err := firebaseAuth.VerifyIDToken(context.Background(), idToken)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), "UID", token.UID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func tokenFromHeader(r *http.Request) string {
	// Get token from authorization header.
	bearer := r.Header.Get("Authorization")
	if len(bearer) > 7 && strings.ToUpper(bearer[0:6]) == "BEARER" {
		return bearer[7:]
	}
	return ""
}

func tokenFromQuery(r *http.Request) string {
	// Get token from query param named "token".
	return r.URL.Query().Get("token")
}

func findToken(r *http.Request, findTokenFns ...func(r *http.Request) string) string {
	var tokenString string

	for _, fn := range findTokenFns {
		tokenString = fn(r)
		if tokenString != "" {
			break
		}
	}

	return tokenString
}
