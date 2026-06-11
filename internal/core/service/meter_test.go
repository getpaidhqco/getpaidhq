package service

import (
	"context"
	"strings"
	"testing"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
)

// fakeMeterRepoSvc embeds the port interface; only Create is exercised here.
type fakeMeterRepoSvc struct {
	port.MeterRepository
	created domain.BillableMetric
	createN int
}

func (r *fakeMeterRepoSvc) Create(_ context.Context, m domain.BillableMetric) (domain.BillableMetric, error) {
	r.created = m
	r.createN++
	return m, nil
}

func TestMeterService_Create(t *testing.T) {
	tests := []struct {
		name    string
		in      port.CreateMeterInput
		wantErr string // substring of the validation error; "" = valid
		stored  bool
	}{
		{
			name:   "count needs no field",
			in:     port.CreateMeterInput{OrgId: "org_1", Code: "api_calls", Name: "API calls", Aggregation: domain.AggregationCount},
			stored: true,
		},
		{
			name:   "sum with field",
			in:     port.CreateMeterInput{OrgId: "org_1", Code: "gb", Name: "GB", Aggregation: domain.AggregationSum, FieldName: "bytes"},
			stored: true,
		},
		{
			name:    "missing code",
			in:      port.CreateMeterInput{OrgId: "org_1", Name: "x", Aggregation: domain.AggregationCount},
			wantErr: "code is required",
		},
		{
			name:    "unknown aggregation",
			in:      port.CreateMeterInput{OrgId: "org_1", Code: "c", Name: "x", Aggregation: domain.AggregationType("nonsense")},
			wantErr: "unknown aggregation",
		},
		{
			name:    "sum without field is rejected",
			in:      port.CreateMeterInput{OrgId: "org_1", Code: "c", Name: "x", Aggregation: domain.AggregationSum},
			wantErr: "field_name is required",
		},
		{
			name:    "bad rounding mode",
			in:      port.CreateMeterInput{OrgId: "org_1", Code: "c", Name: "x", Aggregation: domain.AggregationCount, RoundingMode: "truncate"},
			wantErr: "rounding_mode",
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
			if tt.stored && repo.createN != 1 {
				t.Errorf("expected meter stored once, got %d", repo.createN)
			}
		})
	}
}

func TestValidatePriceConfig(t *testing.T) {
	tier := []domain.PriceTier{{}}
	tests := []struct {
		name      string
		scheme    domain.PriceScheme
		tiers     []domain.PriceTier
		unitCount int
		wantErr   bool
	}{
		{name: "fixed needs no tiers", scheme: domain.Fixed},
		{name: "graduated needs tiers", scheme: domain.Graduated, wantErr: true},
		{name: "graduated with tiers ok", scheme: domain.Graduated, tiers: tier},
		{name: "volume needs tiers", scheme: domain.Volume, wantErr: true},
		{name: "tiered with tiers ok", scheme: domain.Tiered, tiers: tier},
		{name: "fixed allows unit_count", scheme: domain.Fixed, unitCount: 1000},
		{name: "graduated rejects unit_count", scheme: domain.Graduated, tiers: tier, unitCount: 1000, wantErr: true},
		{name: "volume rejects unit_count", scheme: domain.Volume, tiers: tier, unitCount: 2, wantErr: true},
		{name: "tiered with unit_count 1 ok", scheme: domain.Tiered, tiers: tier, unitCount: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePriceConfig(tt.scheme, tt.tiers, tt.unitCount)
			if tt.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
