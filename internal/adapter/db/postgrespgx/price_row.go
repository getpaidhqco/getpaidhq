package postgrespgx

import (
	"time"

	"getpaidhq/internal/core/domain"
)

// priceRow is the postgres on-the-wire shape of a Price. Prices are NOT embedded
// in their variant — composition is a service-layer concern.
//
// Differences from the gorm row (which leans on the driver to coerce NULLs into
// Go zero values, and to coerce the domain's "" enum sentinel back out):
//
//   - category / scheme are NOT NULL enum columns, held as string and converted
//     at the domain boundary (never pass a defined enum type as a pgx arg).
//   - currency is a NOT NULL TEXT column, held as string and converted at the
//     boundary.
//   - billing_interval / trial_interval are NULLABLE enum columns. The domain
//     carries "" for "unset", which is NOT a valid enum member — so the write
//     path maps "" → NULL (nilIfEmpty) and the read path maps NULL → ""
//     (strOrEmpty), mirroring what the gorm driver did implicitly.
//   - label / tax_code / billable_metric_id / filter_field / filter_value are
//     nullable TEXT columns held as *string (NULL ↔ "" via strOrEmpty/nilIfEmpty)
//     so a NULL row scans without error. The gorm row typed them as bare strings.
//   - cycles / billing_interval_qty / trial_interval_qty / min_price /
//     suggested_price are nullable integer columns held as pointer types so a
//     NULL row scans without error. There is no domain "unset" sentinel for these
//     ints (0 is a real value), and the gorm code wrote the zero value straight
//     through, so the write path passes the int directly and the read path maps
//     NULL → 0.
//   - tiers / metadata are nullable JSONB columns mapped via jsonCol; the gorm
//     adapter applied no emptyIfNil to either, so a nil value marshals to JSON
//     null — the pgx jsonCol does the same.
type priceRow struct {
	OrgId              string
	Id                 string
	VariantId          string
	Label              *string
	Category           string
	Scheme             string
	Cycles             *int
	Currency           string
	UnitPrice          int64
	UnitCount          int
	MinPrice           *int64
	SuggestedPrice     *int64
	BillingInterval    *string
	BillingIntervalQty *int
	TrialInterval      *string
	TrialIntervalQty   *int
	TaxCode            *string
	BillableMetricId   *string
	Tiers              jsonCol[[]domain.PriceTier]
	FilterField        *string
	FilterValue        *string
	ProrateOnIncrease  bool
	CreditOnDecrease   bool
	Metadata           jsonCol[map[string]string]
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

const priceColumns = `org_id, id, variant_id, label, category, scheme, cycles, currency, unit_price, unit_count, min_price, suggested_price, billing_interval, billing_interval_qty, trial_interval, trial_interval_qty, tax_code, billable_metric_id, tiers, filter_field, filter_value, prorate_on_increase, credit_on_decrease, metadata, created_at, updated_at`

func (r *priceRow) scanInto(s scanner) error {
	return s.Scan(&r.OrgId, &r.Id, &r.VariantId, &r.Label, &r.Category, &r.Scheme,
		&r.Cycles, &r.Currency, &r.UnitPrice, &r.UnitCount, &r.MinPrice, &r.SuggestedPrice,
		&r.BillingInterval, &r.BillingIntervalQty, &r.TrialInterval, &r.TrialIntervalQty,
		&r.TaxCode, &r.BillableMetricId, &r.Tiers, &r.FilterField, &r.FilterValue,
		&r.ProrateOnIncrease, &r.CreditOnDecrease, &r.Metadata, &r.CreatedAt, &r.UpdatedAt)
}

func (r priceRow) toDomain() domain.Price {
	return domain.Price{
		OrgId:              r.OrgId,
		Id:                 r.Id,
		VariantId:          r.VariantId,
		Label:              strOrEmpty(r.Label),
		Category:           domain.PriceCategory(r.Category),
		Scheme:             domain.PriceScheme(r.Scheme),
		Cycles:             priceIntOrZero(r.Cycles),
		Currency:           domain.Currency(r.Currency),
		UnitPrice:          r.UnitPrice,
		UnitCount:          r.UnitCount,
		MinPrice:           priceInt64OrZero(r.MinPrice),
		SuggestedPrice:     priceInt64OrZero(r.SuggestedPrice),
		BillingInterval:    domain.BillingInterval(strOrEmpty(r.BillingInterval)),
		BillingIntervalQty: priceIntOrZero(r.BillingIntervalQty),
		TrialInterval:      domain.BillingInterval(strOrEmpty(r.TrialInterval)),
		TrialIntervalQty:   priceIntOrZero(r.TrialIntervalQty),
		TaxCode:            strOrEmpty(r.TaxCode),
		BillableMetricId:   strOrEmpty(r.BillableMetricId),
		Tiers:              r.Tiers.V,
		FilterField:        strOrEmpty(r.FilterField),
		FilterValue:        strOrEmpty(r.FilterValue),
		ProrateOnIncrease:  r.ProrateOnIncrease,
		CreditOnDecrease:   r.CreditOnDecrease,
		Metadata:           r.Metadata.V,
		CreatedAt:          r.CreatedAt,
		UpdatedAt:          r.UpdatedAt,
	}
}

func priceRowFromDomain(p domain.Price) priceRow {
	cycles := p.Cycles
	billingIntervalQty := p.BillingIntervalQty
	trialIntervalQty := p.TrialIntervalQty
	minPrice := p.MinPrice
	suggestedPrice := p.SuggestedPrice
	return priceRow{
		OrgId:              p.OrgId,
		Id:                 p.Id,
		VariantId:          p.VariantId,
		Label:              nilIfEmpty(p.Label),
		Category:           string(p.Category),
		Scheme:             string(p.Scheme),
		Cycles:             &cycles,
		Currency:           string(p.Currency),
		UnitPrice:          p.UnitPrice,
		UnitCount:          p.UnitCount,
		MinPrice:           &minPrice,
		SuggestedPrice:     &suggestedPrice,
		BillingInterval:    nilIfEmpty(string(p.BillingInterval)),
		BillingIntervalQty: &billingIntervalQty,
		TrialInterval:      nilIfEmpty(string(p.TrialInterval)),
		TrialIntervalQty:   &trialIntervalQty,
		TaxCode:            nilIfEmpty(p.TaxCode),
		BillableMetricId:   nilIfEmpty(p.BillableMetricId),
		Tiers:              newJSON(p.Tiers),
		FilterField:        nilIfEmpty(p.FilterField),
		FilterValue:        nilIfEmpty(p.FilterValue),
		ProrateOnIncrease:  p.ProrateOnIncrease,
		CreditOnDecrease:   p.CreditOnDecrease,
		Metadata:           newJSON(p.Metadata),
		CreatedAt:          p.CreatedAt,
		UpdatedAt:          p.UpdatedAt,
	}
}

// priceIntOrZero / priceInt64OrZero map a NULL nullable-integer column back to
// 0. The gorm adapter relied on the driver to coerce NULL → zero into its bare
// int fields; pgx requires an explicit pointer scan, so these mirror that on the
// read path. Entity-prefixed to avoid colliding with sibling repos in this
// package.
func priceIntOrZero(v *int) int {
	if v == nil {
		return 0
	}
	return *v
}

func priceInt64OrZero(v *int64) int64 {
	if v == nil {
		return 0
	}
	return *v
}
