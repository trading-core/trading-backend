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

	"github.com/kduong/trading-backend/cmd/authentication-service/internal/user"
	"github.com/kduong/trading-backend/internal/broker"
	"github.com/kduong/trading-backend/internal/eventsource"
	"github.com/kduong/trading-backend/internal/fatal"
	"github.com/kduong/trading-backend/internal/httputil"
	"github.com/kduong/trading-backend/internal/logger"
)

type CreateUserInput struct {
	Email      string `json:"email"`
	Password   string `json:"password"`
	BrokerID   string `json:"broker_id"`
	BrokerType string `json:"broker_type"`
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
	brokerType := strings.ToLower(strings.TrimSpace(input.BrokerType))
	if len(brokerType) == 0 {
		err = merry.Wrap(err).WithHTTPCode(http.StatusBadRequest).WithUserMessage("broker_type is required")
		return
	}
	brokerID := strings.TrimSpace(input.BrokerID)
	if len(brokerID) == 0 {
		err = merry.Wrap(err).WithHTTPCode(http.StatusBadRequest).WithUserMessage("broker_id is required")
		return
	}
	switch brokerType {
	case broker.TypeTastyTrade:
		// supported broker
	default:
		err = merry.Wrap(err).WithHTTPCode(http.StatusBadRequest).WithUserMessage("unsupported broker_type")
		return
	}
	if len(input.Password) < 8 {
		err = merry.Wrap(err).WithHTTPCode(http.StatusBadRequest).WithUserMessage("password must be at least 8 characters")
		return
	}
	_, err = handler.userStore.GetByEmail(ctx, email)
	switch {
	case err == nil:
		err = merry.Wrap(err).WithHTTPCode(http.StatusConflict).WithUserMessage("user already exists")
		return
	case errors.Is(err, user.ErrNotFound):
		err = nil
	default:
		logger.Fatal(err)
		return
	}
	passwordHash, err := HashPassword(input.Password)
	if err != nil {
		return
	}
	object := user.User{
		AccountID:    uuid.NewV4().String(),
		Email:        email,
		PasswordHash: passwordHash,
		CreatedAt:    time.Now().UTC(),
	}
	err = handler.userStore.Put(ctx, object)
	if err != nil {
		if errors.Is(err, user.ErrAlreadyExists) {
			err = merry.Wrap(err).WithHTTPCode(http.StatusConflict).WithUserMessage("user already exists")
			return
		}
		return
	}
	// TODO: check broker credentials and link account if valid
	payload := fatal.UnlessMarshal(EventUserCreated{
		EventBase:  eventsource.NewEventBase(EventTypeUserCreated),
		AccountID:  object.AccountID,
		Email:      object.Email,
		BrokerType: brokerType,
		BrokerID:   brokerID,
	})
	_, err = handler.log.Append(payload)
	fatal.OnError(err)
	httputil.SendResponseJSON(responseWriter, http.StatusCreated, object)
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

const EventTypeUserCreated eventsource.EventType = "user_created"

type EventUserCreated struct {
	eventsource.EventBase
	AccountID  string `json:"account_id"`
	Email      string `json:"email"`
	BrokerType string `json:"broker_type"`
	BrokerID   string `json:"broker_id"`
}
