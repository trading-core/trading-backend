package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/ansel1/merry"
	"github.com/golang-jwt/jwt/v5"
	"github.com/kduong/trading-backend/internal/fatal"
	"github.com/kduong/trading-backend/internal/httputil"
)

func (handler *Handler) RefreshSession(responseWriter http.ResponseWriter, request *http.Request) {
	var err error
	defer func() {
		if err != nil {
			httputil.SendErrorResponse(responseWriter, err)
		}
	}()
	ctx := request.Context()
	authorization := request.Header.Get("Authorization")
	if authorization == "" {
		err = merry.New("missing authorization header").WithHTTPCode(http.StatusUnauthorized).WithUserMessage("unauthorized")
		return
	}
	parts := strings.SplitN(authorization, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		err = merry.New("invalid authorization header").WithHTTPCode(http.StatusUnauthorized).WithUserMessage("unauthorized")
		return
	}
	tokenString := parts[1]
	claims := &jwt.RegisteredClaims{}
	_, err = jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, merry.New("unexpected signing method").WithHTTPCode(http.StatusUnauthorized).WithUserMessage("unauthorized")
		}
		return handler.tokenSecret, nil
	})
	if err != nil {
		err = merry.Wrap(err).WithHTTPCode(http.StatusUnauthorized).WithUserMessage("unauthorized")
		return
	}
	userID := claims.Subject
	object, err := handler.userStore.GetByID(ctx, userID)
	if err != nil {
		err = merry.Wrap(err).WithHTTPCode(http.StatusUnauthorized).WithUserMessage("unauthorized")
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
