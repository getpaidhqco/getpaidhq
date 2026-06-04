package handler

import "getpaidhq/internal/core/port"

// RecordEventResponse is the result of ingesting a usage event.
type RecordEventResponse struct {
	Id     string `json:"id"`
	Status string `json:"status"` // "recorded" | "duplicate"
}

func NewRecordEventResponse(res port.IngestResult) RecordEventResponse {
	status := "recorded"
	if res.Duplicate {
		status = "duplicate"
	}
	return RecordEventResponse{Id: res.Id, Status: status}
}
