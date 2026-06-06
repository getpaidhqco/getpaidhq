package handler

import "getpaidhq/internal/core/port"

// RecordEventResponse is the result of ingesting a usage event.
type RecordEventResponse struct {
	Id     string `json:"id"`
	Status string `json:"status"` // "recorded" | "duplicate" | "accepted" (async, durably queued)
}

func NewRecordEventResponse(res port.IngestResult) RecordEventResponse {
	status := res.Status
	if status == "" {
		status = port.IngestRecorded
	}
	return RecordEventResponse{Id: res.Id, Status: string(status)}
}
