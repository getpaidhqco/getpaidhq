package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/shopspring/decimal"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
	"getpaidhq/internal/lib"
)

// UsageService records usage events and aggregates them. Narrow — no workflow engine.
type UsageService struct {
	meterRepository        port.MeterRepository
	customerRepository     port.CustomerRepository
	subscriptionRepository port.SubscriptionRepository
	orderRepository        port.OrderRepository // subscription → its metered lines (read)
	priceRepository        port.PriceRepository
	ingestor               port.EventIngestor // durable write path (sync = the EventStore; or JetStream)
	eventStore             port.EventStore    // reads / aggregation
	pubsub                 port.PubSub
	logger                 port.Logger
}

func NewUsageService(
	meterRepository port.MeterRepository,
	customerRepository port.CustomerRepository,
	subscriptionRepository port.SubscriptionRepository,
	orderRepository port.OrderRepository,
	priceRepository port.PriceRepository,
	ingestor port.EventIngestor,
	eventStore port.EventStore,
	pubsub port.PubSub,
	logger port.Logger,
) *UsageService {
	return &UsageService{
		meterRepository:        meterRepository,
		customerRepository:     customerRepository,
		subscriptionRepository: subscriptionRepository,
		orderRepository:        orderRepository,
		priceRepository:        priceRepository,
		ingestor:               ingestor,
		eventStore:             eventStore,
		pubsub:                 pubsub,
		logger:                 logger,
	}
}

// MeterUsage is one meter's usage quantity for a subscription over a period.
type MeterUsage struct {
	MetricCode  string
	Aggregation domain.AggregationType
	Quantity    decimal.Decimal
}

// SubscriptionUsage is a subscription's usage for its current billing period.
type SubscriptionUsage struct {
	SubscriptionId     string
	CurrentPeriodStart time.Time
	CurrentPeriodEnd   time.Time
	Meters             []MeterUsage
}

// CurrentPeriodUsage returns a subscription's metered usage for its current
// billing period — the sum, per meter, over the subscription's OWN metered lines
// (the lines stamped with its subscription_id). A pending/cancelled subscription
// (zero period) or one with no metered lines yields an empty Meters slice. An
// unknown subscription surfaces as port.ErrNotFound for the caller to map to 404.
func (s *UsageService) CurrentPeriodUsage(ctx context.Context, orgId, subscriptionId string) (SubscriptionUsage, error) {
	sub, err := s.subscriptionRepository.FindById(ctx, orgId, subscriptionId)
	if err != nil {
		return SubscriptionUsage{}, err
	}
	out := SubscriptionUsage{
		SubscriptionId:     sub.Id,
		CurrentPeriodStart: sub.CurrentPeriodStart,
		CurrentPeriodEnd:   sub.CurrentPeriodEnd,
		Meters:             []MeterUsage{},
	}
	if sub.CurrentPeriodStart.IsZero() {
		return out, nil
	}
	items, err := s.orderRepository.FindOrderItemsBySubscriptionId(ctx, orgId, sub.Id)
	if err != nil {
		return SubscriptionUsage{}, err
	}
	for _, it := range items {
		price, perr := s.priceRepository.FindById(ctx, orgId, it.PriceId)
		if perr != nil {
			return SubscriptionUsage{}, perr
		}
		if !price.IsMetered() {
			continue
		}
		units, uerr := s.UsageForSubscription(ctx, sub, price, sub.CurrentPeriodStart, sub.CurrentPeriodEnd)
		if uerr != nil {
			return SubscriptionUsage{}, uerr
		}
		metric, merr := s.meterRepository.FindById(ctx, orgId, price.BillableMetricId)
		if merr != nil {
			return SubscriptionUsage{}, merr
		}
		out.Meters = append(out.Meters, MeterUsage{MetricCode: metric.Code, Aggregation: metric.Aggregation, Quantity: units})
	}
	return out, nil
}

// RecordEvent validates + stores one usage event. Returns the ingest result
// (Duplicate=true when a resend with the same external_id was ignored).
func (s *UsageService) RecordEvent(ctx context.Context, in port.RecordEventInput) (port.IngestResult, error) {
	metric, err := s.meterRepository.FindByCode(ctx, in.OrgId, in.MetricCode)
	if err != nil {
		return port.IngestResult{}, lib.NewCustomError(lib.BadRequestError, "unknown metric code", err)
	}

	// Identify the customer (§6 step 2). Exactly one id is required. A customer_id
	// must exist; an unknown external_customer_id is accepted as-is (orphan event,
	// attached later if a customer with that external id is created). When the
	// external id resolves now, we also store the internal customer_id.
	if in.CustomerId == "" && in.ExternalCustomerId == "" {
		return port.IngestResult{}, lib.NewCustomError(lib.BadRequestError, "customer_id or external_customer_id is required", nil)
	}
	if in.CustomerId != "" {
		if _, cerr := s.customerRepository.FindById(ctx, in.OrgId, in.CustomerId); cerr != nil {
			if errors.Is(cerr, port.ErrNotFound) {
				return port.IngestResult{}, lib.NewCustomError(lib.NotFoundError, "customer not found", cerr)
			}
			return port.IngestResult{}, cerr
		}
	} else if cust, cerr := s.customerRepository.FindByExternalId(ctx, in.OrgId, in.ExternalCustomerId); cerr == nil {
		in.CustomerId = cust.Id
	} else if !errors.Is(cerr, port.ErrNotFound) {
		return port.IngestResult{}, cerr
	}

	// Attribution (§6 step 3). If a subscription is named it must belong to the
	// customer and carry a metered price for this metric.
	if in.SubscriptionId != "" {
		if in.CustomerId == "" {
			return port.IngestResult{}, lib.NewCustomError(lib.BadRequestError, "subscription_id requires a known customer", nil)
		}
		metered, merr := s.subscriptionRepository.FindActiveMeteredForMeter(ctx, in.OrgId, in.CustomerId, metric.Id)
		if merr != nil {
			return port.IngestResult{}, merr
		}
		if !containsSubscription(metered, in.SubscriptionId) {
			return port.IngestResult{}, lib.NewCustomError(lib.BadRequestError, "subscription does not carry a metered price for this metric", nil)
		}
	}

	value := decimal.Zero
	if metric.Aggregation != domain.AggregationCount && metric.Aggregation != domain.AggregationUniqueCount {
		raw, ok := in.Metadata[metric.FieldName]
		if !ok {
			return port.IngestResult{}, lib.NewCustomError(lib.BadRequestError, "metadata is missing the metric field "+metric.FieldName, nil)
		}
		v, perr := decimal.NewFromString(raw)
		if perr != nil {
			return port.IngestResult{}, lib.NewCustomError(lib.BadRequestError, "metric field "+metric.FieldName+" is not numeric", perr)
		}
		value = v
	}

	ts := in.Timestamp
	if ts.IsZero() {
		ts = time.Now().UTC()
	}

	event := domain.MeterEvent{
		OrgId:              in.OrgId,
		Id:                 lib.GenerateId("mev"),
		CustomerId:         in.CustomerId,
		ExternalCustomerId: in.ExternalCustomerId,
		MetricCode:         in.MetricCode,
		SubscriptionId:     in.SubscriptionId,
		ExternalId:         in.ExternalId,
		Metadata:           in.Metadata,
		Value:              value,
		Timestamp:          ts,
		CreatedAt:          time.Now().UTC(),
	}

	res, err := s.ingestor.Ingest(ctx, event)
	if err != nil {
		// Single-handling rule: wrap and return; the HTTP boundary logs it.
		return port.IngestResult{}, fmt.Errorf("ingest usage event: %w", err)
	}
	_ = s.pubsub.Publish(in.OrgId, "usage.recorded", event)
	return res, nil
}

// containsSubscription reports whether subId is one of subs.
func containsSubscription(subs []domain.Subscription, subId string) bool {
	for _, s := range subs {
		if s.Id == subId {
			return true
		}
	}
	return false
}

// AggregateForPeriod adds up usage for the given query using the metric's aggregation,
// then applies the metric's rounding. The caller builds the UsageQuery (customer +
// window + optional subscription attribution).
func (s *UsageService) AggregateForPeriod(ctx context.Context, metric domain.BillableMetric, q port.UsageQuery) (decimal.Decimal, error) {
	q.FieldName = metric.FieldName
	q.MetricCode = metric.Code

	var (
		units decimal.Decimal
		err   error
	)
	switch metric.Aggregation {
	case domain.AggregationCount:
		var n int64
		n, err = s.eventStore.Count(ctx, q)
		units = decimal.NewFromInt(n)
	case domain.AggregationUniqueCount:
		var n int64
		n, err = s.eventStore.UniqueCount(ctx, q)
		units = decimal.NewFromInt(n)
	case domain.AggregationSum:
		units, err = s.eventStore.Sum(ctx, q)
	case domain.AggregationMax:
		units, err = s.eventStore.Max(ctx, q)
	case domain.AggregationLatest:
		units, err = s.eventStore.Latest(ctx, q)
	case domain.AggregationWeightedSum:
		units, err = s.eventStore.WeightedSum(ctx, q, decimal.Zero)
	default:
		return decimal.Zero, errors.New("unknown aggregation type: " + string(metric.Aggregation))
	}
	if err != nil {
		return decimal.Zero, err
	}
	return applyRounding(metric, units), nil
}

// UsageForSubscription aggregates a metered subscription's usage for [from, to),
// resolving the meter from the price. It sums events attributed to this subscription
// and — only when this is the customer's earliest active metered subscription for the
// meter — the unattributed events too (§10), so a catch-all is billed exactly once
// across multiple metered subs. It also fills the customer's external id so usage
// recorded against external_customer_id before the customer existed is still matched (§8).
func (s *UsageService) UsageForSubscription(ctx context.Context, sub domain.Subscription, price domain.Price, from, to time.Time) (decimal.Decimal, error) {
	metric, err := s.meterRepository.FindById(ctx, sub.OrgId, price.BillableMetricId)
	if err != nil {
		return decimal.Zero, err
	}

	q := port.UsageQuery{
		OrgId:          sub.OrgId,
		CustomerId:     sub.CustomerId,
		From:           from,
		To:             to,
		SubscriptionId: sub.Id,
	}

	// Match events sent against the merchant's own customer id, not just ours.
	if cust, cerr := s.customerRepository.FindById(ctx, sub.OrgId, sub.CustomerId); cerr == nil {
		q.ExternalCustomerId = cust.ExternalId
	} else if !errors.Is(cerr, port.ErrNotFound) {
		return decimal.Zero, cerr
	}

	// The earliest metered sub for (customer, meter) is the catch-all for
	// unattributed usage.
	metered, err := s.subscriptionRepository.FindActiveMeteredForMeter(ctx, sub.OrgId, sub.CustomerId, metric.Id)
	if err != nil {
		return decimal.Zero, err
	}
	q.IncludeUnattributed = len(metered) > 0 && metered[0].Id == sub.Id

	return s.AggregateForPeriod(ctx, metric, q)
}

func applyRounding(metric domain.BillableMetric, units decimal.Decimal) decimal.Decimal {
	scale := int32(metric.RoundingScale)
	switch metric.RoundingMode {
	case "ceil":
		return units.RoundCeil(scale)
	case "floor":
		return units.RoundFloor(scale)
	case "round":
		return units.Round(scale)
	default:
		return units
	}
}
