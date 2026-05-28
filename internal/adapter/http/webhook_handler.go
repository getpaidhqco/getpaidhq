package handler

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/go-fuego/fuego"
	"github.com/go-fuego/fuego/option"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
	"getpaidhq/internal/core/service"
)

// WebhookHandler handles incoming payment webhook requests.
type WebhookHandler struct {
	webhookService *service.WebhookService
	logger         port.Logger
}

func NewWebhookHandler(service *service.WebhookService, logger port.Logger) *WebhookHandler {
	return &WebhookHandler{webhookService: service, logger: logger}
}

// RegisterRoutes registers the raw-body webhook endpoint. PSPs sign the
// unparsed body, so the route is wired through PostStd which skips
// Fuego's JSON binder.
func (u *WebhookHandler) RegisterRoutes(s *fuego.Server) {
	fuego.PostStd(s, "/notify", u.Process,
		option.Tags("Webhooks"),
		option.Summary("PSP webhook receiver"),
		option.Query("p", "PSP identifier"),
	)
}

func (u *WebhookHandler) Process(w http.ResponseWriter, r *http.Request) {
	psp := r.URL.Query().Get("p")

	// Read the raw body. PSPs sign the unparsed bytes; a transient read
	// failure that we silently swallow would surface as a signature
	// verification failure against empty data, indistinguishable in logs
	// from a forged request. Surface the I/O failure as a 400 so the PSP
	// retries instead of treating an empty-body 200 as success.
	jsonData, err := io.ReadAll(r.Body)
	if err != nil {
		u.logger.Error("webhook body read failed", "error", err.Error(), "psp", psp)
		writeWebhookError(w, http.StatusBadRequest, "could not read request body")
		return
	}

	u.logger.Debug("Processing webhook")
	if err := u.webhookService.HandlePaymentWebhook(r.Context(), port.PaymentWebhookPayload{
		Psp:  domain.Gateway(psp),
		Data: string(jsonData),
	}); err != nil {
		u.logger.Errorf("Error processing webhook: %s", err.Error())
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

// writeWebhookError emits a minimal JSON envelope for the raw-body webhook
// endpoint. PSPs typically only check status code and retry on 4xx/5xx, so
// the body shape is informational; we keep it consistent with the success
// envelope (a single string field) rather than the full ApiError shape.
func writeWebhookError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": message})
}
