package service

import (
	"context"
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
		wantErr bool
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
			wantErr: true,
		},
		{
			name:    "unknown aggregation",
			in:      port.CreateMeterInput{OrgId: "org_1", Code: "c", Name: "x", Aggregation: domain.AggregationType("nonsense")},
			wantErr: true,
		},
		{
			name:    "sum without field is rejected",
			in:      port.CreateMeterInput{OrgId: "org_1", Code: "c", Name: "x", Aggregation: domain.AggregationSum},
			wantErr: true,
		},
		{
			name:    "bad rounding mode",
			in:      port.CreateMeterInput{OrgId: "org_1", Code: "c", Name: "x", Aggregation: domain.AggregationCount, RoundingMode: "truncate"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeMeterRepoSvc{}
			svc := NewMeterService(repo, &recordingPubSub{}, silentLogger{})
			_, err := svc.Create(context.Background(), tt.in)
			if tt.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.stored && repo.createN != 1 {
				t.Errorf("expected meter stored once, got %d", repo.createN)
			}
			if tt.wantErr && repo.createN != 0 {
				t.Errorf("invalid input must not store, got %d writes", repo.createN)
			}
		})
	}
}

func TestValidatePriceConfig(t *testing.T) {
	tier := []domain.PriceTier{{}}
	tests := []struct {
		name    string
		scheme  domain.PriceScheme
		tiers   []domain.PriceTier
		wantErr bool
	}{
		{name: "fixed needs no tiers", scheme: domain.Fixed},
		{name: "graduated needs tiers", scheme: domain.Graduated, wantErr: true},
		{name: "graduated with tiers ok", scheme: domain.Graduated, tiers: tier},
		{name: "volume needs tiers", scheme: domain.Volume, wantErr: true},
		{name: "tiered with tiers ok", scheme: domain.Tiered, tiers: tier},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePriceConfig(tt.scheme, tt.tiers)
			if tt.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
