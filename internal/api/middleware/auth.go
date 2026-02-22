package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"vwap/internal/auth"
	"vwap/internal/httpwrap"
)

const (
	headerAuthorization = "Authorization"

	bearerPrefix = "Bearer "

	queryToken = "token"
)

type ctxKeyAuthUserID struct{}

func AuthMiddleware(authService auth.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get(headerAuthorization)
			token := strings.TrimPrefix(authHeader, bearerPrefix)

			if token == "" {
				token = r.URL.Query().Get(queryToken)
			}

			ctx := r.Context()

			tokenInfo, err := authService.ValidateToken(ctx, token)
			if err != nil {
				if errors.Is(err, auth.ErrTokenExpired) || errors.Is(err, auth.ErrTokenNotFound) {
					errResp := httpwrap.NewUnauthorizedError(err)

					errResp.Render(w, r)

					return
				}

				errResp := httpwrap.NewInternalServerError(err)

				errResp.Render(w, r)

				return
			}

			ctx = SetUserID(ctx, tokenInfo.UserID)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GetUserID(r *http.Request) uuid.UUID {
	userID, _ := r.Context().Value(ctxKeyAuthUserID{}).(uuid.UUID)

	return userID
}

func SetUserID(ctx context.Context, userID uuid.UUID) context.Context {
	return context.WithValue(ctx, ctxKeyAuthUserID{}, userID)
}
