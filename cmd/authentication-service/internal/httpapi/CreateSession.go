package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/ansel1/merry"
	"github.com/kduong/trading-backend/cmd/authentication-service/internal/userstore"
	"github.com/kduong/trading-backend/internal/fatal"
	"github.com/kduong/trading-backend/internal/httpx"
	"golang.org/x/crypto/bcrypt"
)

type CreateSessionInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type CreateSessionOutput struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresAt   string `json:"expires_at"`
	UserID      string `json:"user_id"`
	Email       string `json:"email"`
}

func (handler *Handler) CreateSession(responseWriter http.ResponseWriter, request *http.Request) {
	var err error
	defer func() {
		if err != nil {
			httpx.SendErrorResponse(responseWriter, err)
		}
	}()
	ctx := request.Context()
	var input CreateSessionInput
	err = json.NewDecoder(request.Body).Decode(&input)
	if err != nil {
		err = merry.Wrap(err).WithHTTPCode(http.StatusBadRequest).WithUserMessage("invalid request body")
		return
	}
	input.Email = strings.ToLower(strings.TrimSpace(input.Email))
	if len(input.Email) == 0 || len(input.Password) == 0 {
		err = merry.Wrap(err).WithHTTPCode(http.StatusBadRequest).WithUserMessage("email and password are required")
		return
	}
	object, err := handler.userStore.GetByEmail(ctx, input.Email)
	if err != nil {
		if errors.Is(err, userstore.ErrNotFound) {
			err = merry.Wrap(err).WithHTTPCode(http.StatusUnauthorized).WithUserMessage("invalid credentials")
			return
		}
		return
	}
	isPasswordValid := VerifyPassword(input.Password, object.PasswordHash)
	if !isPasswordValid {
		err = merry.Wrap(err).WithHTTPCode(http.StatusUnauthorized).WithUserMessage("invalid credentials")
		return
	}
	token, expiresAt, err := handler.GenerateToken(object)
	fatal.OnError(err)
	output := CreateSessionOutput{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresAt:   expiresAt.Format(time.RFC3339),
		UserID:      object.ID,
		Email:       object.Email,
	}
	responseWriter.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(responseWriter).Encode(&output)
	fatal.OnErrorUnlessDone(ctx, err)
}

func VerifyPassword(password string, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
