package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/ansel1/merry"
	"github.com/kduong/trading-backend/cmd/auth-service/internal/auth"
	"github.com/kduong/trading-backend/internal/account"
	"github.com/kduong/trading-backend/internal/httputil"
)

type LoginInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginOutput struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresAt   string `json:"expires_at"`
	AccountID   string `json:"account_id"`
	Email       string `json:"email"`
}

func (handler *Handler) Login(responseWriter http.ResponseWriter, request *http.Request) {
	var err error
	defer func() {
		if err != nil {
			httputil.SendErrorResponse(responseWriter, err)
		}
	}()
	ctx := request.Context()
	var input LoginInput
	err = json.NewDecoder(request.Body).Decode(&input)
	if err != nil {
		err = merry.Wrap(err).WithHTTPCode(http.StatusBadRequest).WithUserMessage("invalid request body")
		return
	}
	input.Email = strings.ToLower(strings.TrimSpace(input.Email))
	if len(input.Email) == 0 || len(input.Password) == 0 {
		err = merry.New("email and password are required").WithHTTPCode(http.StatusBadRequest).WithUserMessage("email and password are required")
		return
	}
	object, err := handler.accountStore.GetByEmail(ctx, input.Email)
	if errors.Is(err, account.ErrAccountNotFound) {
		err = merry.New("invalid credentials").WithHTTPCode(http.StatusUnauthorized).WithUserMessage("invalid credentials")
		return
	}
	if err != nil {
		return
	}
	if !auth.VerifyPassword(input.Password, object.PasswordHash) {
		err = merry.New("invalid credentials").WithHTTPCode(http.StatusUnauthorized).WithUserMessage("invalid credentials")
		return
	}
	token, expiresAt, err := handler.tokenManager.GenerateToken(object)
	if err != nil {
		return
	}
	httputil.SendResponseJSON(responseWriter, http.StatusOK, LoginOutput{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresAt:   expiresAt.Format("2006-01-02T15:04:05Z07:00"),
		AccountID:   object.AccountID,
		Email:       object.Email,
	})
}
