package httpapi

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/gorilla/mux"
	"github.com/kduong/trading-backend/internal/auth"
	"github.com/kduong/trading-backend/internal/httputil"
)

type Bot struct {
	ID        string `json:"id"`
	AccountID string `json:"account_id"`
	Name      string `json:"name"`
	Status    string `json:"status"` // "running" or "stopped"
	CreatedAt string `json:"created_at"`
}

type Handler struct {
	botsMutex sync.RWMutex
	bots      map[string]Bot
}

type NewRouterInput struct {
	AuthMiddleware *auth.Middleware
}

func NewRouter(input NewRouterInput) *mux.Router {
	handler := &Handler{
		bots: make(map[string]Bot),
	}
	router := mux.NewRouter().StrictSlash(true)
	botV1Router := router.PathPrefix("/bots/v1").Subrouter()
	botV1Router.Use(input.AuthMiddleware.Handle)

	botV1Router.HandleFunc("/running", handler.ListRunningBotsByAccount).Methods(http.MethodGet).Name("ListRunningBotsByAccount")
	botV1Router.HandleFunc("/{id}/start", handler.StartBot).Methods(http.MethodPost).Name("StartBot")
	botV1Router.HandleFunc("/{id}/stop", handler.StopBot).Methods(http.MethodPost).Name("StopBot")

	return router
}

func (h *Handler) ListRunningBotsByAccount(w http.ResponseWriter, r *http.Request) {
	accountID := r.Header.Get("X-Account-ID")

	h.botsMutex.RLock()
	defer h.botsMutex.RUnlock()

	var runningBots []Bot
	for _, bot := range h.bots {
		if bot.AccountID == accountID && bot.Status == "running" {
			runningBots = append(runningBots, bot)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(runningBots)
}

func (h *Handler) StartBot(w http.ResponseWriter, r *http.Request) {
	botID := mux.Vars(r)["id"]
	accountID := r.Header.Get("X-Account-ID")

	h.botsMutex.Lock()
	defer h.botsMutex.Unlock()

	bot, exists := h.bots[botID]
	if !exists {
		httputil.SendResponse(w, http.StatusNotFound, map[string]string{
			"error": "bot not found",
		})
		return
	}

	if bot.AccountID != accountID {
		httputil.SendResponse(w, http.StatusForbidden, map[string]string{
			"error": "unauthorized",
		})
		return
	}

	bot.Status = "running"
	h.bots[botID] = bot

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(bot)
}

func (h *Handler) StopBot(w http.ResponseWriter, r *http.Request) {
	botID := mux.Vars(r)["id"]
	accountID := r.Header.Get("X-Account-ID")

	h.botsMutex.Lock()
	defer h.botsMutex.Unlock()

	bot, exists := h.bots[botID]
	if !exists {
		httputil.SendResponse(w, http.StatusNotFound, map[string]string{
			"error": "bot not found",
		})
		return
	}

	if bot.AccountID != accountID {
		httputil.SendResponse(w, http.StatusForbidden, map[string]string{
			"error": "unauthorized",
		})
		return
	}

	bot.Status = "stopped"
	h.bots[botID] = bot

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(bot)
}
