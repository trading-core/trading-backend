package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/ansel1/merry"
	"github.com/kduong/trading-backend/cmd/authentication-service/internal/user"
	"github.com/kduong/trading-backend/internal/fatal"
	"github.com/kduong/trading-backend/internal/httputil"
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
	AccountID   string `json:"account_id"`
	Email       string `json:"email"`
}

func (handler *Handler) CreateSession(responseWriter http.ResponseWriter, request *http.Request) {
	var err error
	defer func() {
		if err != nil {
			httputil.SendErrorResponse(responseWriter, err)
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
		if errors.Is(err, user.ErrNotFound) {
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
		ExpiresAt:   expiresAt.Format("2006-01-02T15:04:05Z07:00"),
		AccountID:   object.AccountID,
		Email:       object.Email,
	}
	httputil.SendResponseJSON(responseWriter, http.StatusOK, output)
}

func VerifyPassword(password string, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
