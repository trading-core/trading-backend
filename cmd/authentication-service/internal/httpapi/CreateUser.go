package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/ansel1/merry"
	uuid "github.com/satori/go.uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/kduong/trading-backend/cmd/authentication-service/internal/userstore"
	"github.com/kduong/trading-backend/internal/fatal"
	"github.com/kduong/trading-backend/internal/httputil"
)

type CreateUserInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (handler *Handler) CreateUser(responseWriter http.ResponseWriter, request *http.Request) {
	var err error
	defer func() {
		if err != nil {
			httputil.SendErrorResponse(responseWriter, err)
		}
	}()
	ctx := request.Context()
	var input CreateUserInput
	err = json.NewDecoder(request.Body).Decode(&input)
	if err != nil {
		err = merry.Wrap(err).WithHTTPCode(http.StatusBadRequest).WithUserMessage("invalid request body")
		return
	}
	email := strings.ToLower(strings.TrimSpace(input.Email))
	if len(email) == 0 {
		err = merry.Wrap(err).WithHTTPCode(http.StatusBadRequest).WithUserMessage("email is required")
		return
	}
	passwordHash, err := HashPassword(input.Password)
	if err != nil {
		return
	}
	object := userstore.User{
		ID:           uuid.NewV4().String(),
		Email:        email,
		PasswordHash: passwordHash,
		CreatedAt:    time.Now().UTC(),
	}
	err = handler.userStore.Put(ctx, object)
	if err != nil {
		if errors.Is(err, userstore.ErrAlreadyExists) {
			err = merry.Wrap(err).WithHTTPCode(http.StatusConflict).WithUserMessage("user already exists")
			return
		}
		return
	}
	responseWriter.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(responseWriter).Encode(&object)
	fatal.OnErrorUnlessDone(ctx, err)
}

func HashPassword(password string) (hashPassword string, err error) {
	hashBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return
	}
	hashPassword = string(hashBytes)
	return
}
