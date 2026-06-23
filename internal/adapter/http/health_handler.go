package handler

import (
	"github.com/go-fuego/fuego"
	"github.com/go-fuego/fuego/option"

	"getpaidhq/internal/core/port"
)

// HealthHandler handles health check requests.
type HealthHandler struct {
	logger port.Logger
}

func NewHealthHandler(logger port.Logger) *HealthHandler {
	return &HealthHandler{logger: logger}
}

func (u *HealthHandler) RegisterRoutes(s *fuego.Server) {
	fuego.Get(s, "/health", u.Healthcheck,
		option.Summary("Liveness probe"),
		option.Tags("Health"),
		option.OperationID("getHealth"),
	)
}

type HealthResponse struct {
	Status string `json:"status"`
}

func (u *HealthHandler) Healthcheck(_ fuego.ContextNoBody) (HealthResponse, error) {
	return HealthResponse{Status: "ok"}, nil
}
