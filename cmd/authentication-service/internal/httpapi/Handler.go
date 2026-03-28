package httpapi

import (
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/mux"

	"github.com/kduong/trading-backend/cmd/authentication-service/internal/userstore"
)

type Handler struct {
	userStore   userstore.Store
	tokenSecret []byte
	expiryTTL   time.Duration
}

type NewRouterInput struct {
	UserStore   userstore.Store
	TokenSecret []byte
	ExpiryTTL   time.Duration
}

func NewRouter(input NewRouterInput) *mux.Router {
	handler := &Handler{
		userStore:   input.UserStore,
		tokenSecret: input.TokenSecret,
		expiryTTL:   input.ExpiryTTL,
	}
	router := mux.NewRouter().StrictSlash(true)
	authV1Router := router.PathPrefix("/auth/v1").Subrouter()
	authV1Router.HandleFunc("/users", handler.CreateUser).Methods(http.MethodPost).Name("CreateUser")
	authV1Router.HandleFunc("/sessions", handler.CreateSession).Methods(http.MethodPost).Name("CreateSession")
	authV1Router.HandleFunc("/sessions/refresh", handler.RefreshSession).Methods(http.MethodPost).Name("RefreshSession")
	return router
}

func (handler *Handler) GenerateToken(user *userstore.User) (string, time.Time, error) {
	now := time.Now().UTC()
	expiresAt := now.Add(handler.expiryTTL)
	claims := jwt.RegisteredClaims{
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(expiresAt),
		Subject:   user.ID,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(handler.tokenSecret)
	if err != nil {
		return "", time.Time{}, err
	}
	return signed, expiresAt, nil
}
