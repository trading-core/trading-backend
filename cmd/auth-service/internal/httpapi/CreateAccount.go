package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/ansel1/merry"
	"github.com/kduong/trading-backend/cmd/auth-service/internal/auth"
	"github.com/kduong/trading-backend/internal/account"
	"github.com/kduong/trading-backend/internal/fatal"
	"github.com/kduong/trading-backend/internal/httputil"
)

type CreateAccountInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (handler *Handler) CreateAccount(responseWriter http.ResponseWriter, request *http.Request) {
	var err error
	defer func() {
		if err != nil {
			httputil.SendErrorResponse(responseWriter, err)
		}
	}()
	ctx := request.Context()
	var input CreateAccountInput
	err = json.NewDecoder(request.Body).Decode(&input)
	if err != nil {
		err = merry.Wrap(err).WithHTTPCode(http.StatusBadRequest).WithUserMessage("invalid request body")
		return
	}
	input.Email = strings.ToLower(strings.TrimSpace(input.Email))
	if len(input.Email) == 0 {
		err = merry.New("email is required").WithHTTPCode(http.StatusBadRequest).WithUserMessage("email is required")
		return
	}
	if len(input.Password) < 8 {
		err = merry.New("password must be at least 8 characters").WithHTTPCode(http.StatusBadRequest).WithUserMessage("password must be at least 8 characters")
		return
	}
	_, err = handler.accountStore.GetByEmail(ctx, input.Email)
	if err == nil {
		err = merry.New("account already exists").WithHTTPCode(http.StatusConflict).WithUserMessage("account already exists")
		return
	}
	if !errors.Is(err, account.ErrAccountNotFound) {
		return
	}
	passwordHash, err := auth.HashPassword(input.Password)
	if err != nil {
		return
	}
	accountID, err := auth.GenerateAccountID()
	if err != nil {
		return
	}
	object := &account.Object{
		AccountID:    accountID,
		Email:        input.Email,
		PasswordHash: passwordHash,
		CreatedAt:    time.Now().UTC(),
	}
	err = handler.accountStore.Put(ctx, object)
	if errors.Is(err, account.ErrAccountAlreadyExists) {
		err = merry.New("account already exists").WithHTTPCode(http.StatusConflict).WithUserMessage("account already exists")
		return
	}
	if err != nil {
		return
	}
	httputil.SendResponseJSON(responseWriter, http.StatusCreated, object)
	fatal.OnErrorUnlessDone(ctx, err)
}
