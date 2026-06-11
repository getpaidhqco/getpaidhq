package service

import (
	"context"
	"strings"
	"testing"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
)

// Carry-over meters allow only the standing-level aggregations and no
// filters/group_by; weighted_sum exists only on carry-over meters
// (stock-billing-architecture-impact.md §4).
func TestMeterService_Create_CarryOver(t *testing.T) {
	tests := []struct {
		name    string
		in      port.CreateMeterInput
		wantErr string // substring of the validation error; "" = valid
	}{
		{
			name: "latest is valid",
			in:   port.CreateMeterInput{OrgId: "org_1", Code: "seats", Name: "Seats", Aggregation: domain.AggregationLatest, FieldName: "seat_id", CarryOver: true},
		},
		{
			name: "max is valid",
			in:   port.CreateMeterInput{OrgId: "org_1", Code: "seats", Name: "Seats", Aggregation: domain.AggregationMax, FieldName: "seat_id", CarryOver: true},
		},
		{
			name: "unique_count is valid",
			in:   port.CreateMeterInput{OrgId: "org_1", Code: "seats", Name: "Seats", Aggregation: domain.AggregationUniqueCount, FieldName: "seat_id", CarryOver: true},
		},
		{
			name: "weighted_sum is valid",
			in:   port.CreateMeterInput{OrgId: "org_1", Code: "seats", Name: "Seats", Aggregation: domain.AggregationWeightedSum, FieldName: "seat_id", CarryOver: true},
		},
		{
			name:    "count is rejected",
			in:      port.CreateMeterInput{OrgId: "org_1", Code: "seats", Name: "Seats", Aggregation: domain.AggregationCount, CarryOver: true},
			wantErr: "not supported for carry-over",
		},
		{
			name:    "sum is rejected",
			in:      port.CreateMeterInput{OrgId: "org_1", Code: "seats", Name: "Seats", Aggregation: domain.AggregationSum, FieldName: "n", CarryOver: true},
			wantErr: "not supported for carry-over",
		},
		{
			name:    "weighted_sum without carry_over is rejected",
			in:      port.CreateMeterInput{OrgId: "org_1", Code: "avg_gb", Name: "Avg GB", Aggregation: domain.AggregationWeightedSum, FieldName: "gb"},
			wantErr: "weighted_sum requires carry_over",
		},
		{
			name: "filters are rejected",
			in: port.CreateMeterInput{OrgId: "org_1", Code: "seats", Name: "Seats", Aggregation: domain.AggregationWeightedSum, FieldName: "seat_id", CarryOver: true,
				Filters: []domain.MetricFilter{{Field: "type", Values: []string{"x"}}}},
			wantErr: "filters are not supported",
		},
		{
			name: "group_by is rejected",
			in: port.CreateMeterInput{OrgId: "org_1", Code: "seats", Name: "Seats", Aggregation: domain.AggregationWeightedSum, FieldName: "seat_id", CarryOver: true,
				GroupBy: []string{"team"}},
			wantErr: "group_by is not supported",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeMeterRepoSvc{}
			svc := NewMeterService(repo, &recordingPubSub{}, silentLogger{})
			_, err := svc.Create(context.Background(), tt.in)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error = %q, want it to contain %q", err.Error(), tt.wantErr)
				}
				if repo.createN != 0 {
					t.Errorf("invalid input must not store, got %d writes", repo.createN)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
