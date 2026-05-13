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
	jsonData, _ := io.ReadAll(r.Body)
	psp := r.URL.Query().Get("p")

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
