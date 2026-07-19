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
	event, err := s.buildEvent(ctx, in, map[string]domain.BillableMetric{})
	if err != nil {
		return port.IngestResult{}, err
	}
	res, err := s.ingestor.Ingest(ctx, event)
	if err != nil {
		// Single-handling rule: wrap and return; the HTTP boundary logs it.
		return port.IngestResult{}, fmt.Errorf("ingest usage event: %w", err)
	}
	_ = s.pubsub.Publish(ctx, in.OrgId, "usage.recorded", event)
	return res, nil
}

// RecordEvents validates and stores a batch of usage events. Each event is
// processed independently: a validation failure yields a Rejected result (with
// the reason) for that event without affecting the others, so the result slice
// aligns 1:1 with inputs. Meter lookups are cached across the batch. An ingest
// (infrastructure) failure aborts the batch and is returned as an error.
func (s *UsageService) RecordEvents(ctx context.Context, inputs []port.RecordEventInput) ([]port.IngestResult, error) {
	results := make([]port.IngestResult, len(inputs))
	meterCache := make(map[string]domain.BillableMetric)
	for i, in := range inputs {
		event, verr := s.buildEvent(ctx, in, meterCache)
		if verr != nil {
			results[i] = port.IngestResult{Status: port.IngestRejected, Error: verr.Error()}
			continue
		}
		res, err := s.ingestor.Ingest(ctx, event)
		if err != nil {
			return nil, fmt.Errorf("ingest usage event: %w", err)
		}
		_ = s.pubsub.Publish(ctx, in.OrgId, "usage.recorded", event)
		results[i] = res
	}
	return results, nil
}

// buildEvent validates one input and constructs the MeterEvent to ingest. The
// meterCache avoids repeated meter lookups when called across a batch.
func (s *UsageService) buildEvent(ctx context.Context, in port.RecordEventInput, meterCache map[string]domain.BillableMetric) (domain.MeterEvent, error) {
	metric, ok := meterCache[in.MetricCode]
	if !ok {
		m, err := s.meterRepository.FindByCode(ctx, in.OrgId, in.MetricCode)
		if err != nil {
			return domain.MeterEvent{}, lib.NewCustomError(lib.BadRequestError, "unknown metric code", err)
		}
		metric = m
		meterCache[in.MetricCode] = m
	}

	// Identify the customer (§6 step 2). Exactly one id is required. A customer_id
	// must exist; an unknown external_customer_id is accepted as-is (orphan event,
	// attached later if a customer with that external id is created). When the
	// external id resolves now, we also store the internal customer_id.
	if in.CustomerId == "" && in.ExternalCustomerId == "" {
		return domain.MeterEvent{}, lib.NewCustomError(lib.BadRequestError, "customer_id or external_customer_id is required", nil)
	}
	if in.CustomerId != "" {
		if _, cerr := s.customerRepository.FindById(ctx, in.OrgId, in.CustomerId); cerr != nil {
			if errors.Is(cerr, port.ErrNotFound) {
				return domain.MeterEvent{}, lib.NewCustomError(lib.NotFoundError, "customer not found", cerr)
			}
			return domain.MeterEvent{}, cerr
		}
	} else if cust, cerr := s.customerRepository.FindByExternalId(ctx, in.OrgId, in.ExternalCustomerId); cerr == nil {
		in.CustomerId = cust.Id
	} else if !errors.Is(cerr, port.ErrNotFound) {
		return domain.MeterEvent{}, cerr
	}

	// Attribution (§6 step 3). If a subscription is named it must belong to the
	// customer and carry a metered price for this metric.
	if in.SubscriptionId != "" {
		if in.CustomerId == "" {
			return domain.MeterEvent{}, lib.NewCustomError(lib.BadRequestError, "subscription_id requires a known customer", nil)
		}
		metered, merr := s.subscriptionRepository.FindActiveMeteredForMeter(ctx, in.OrgId, in.CustomerId, metric.Id)
		if merr != nil {
			return domain.MeterEvent{}, merr
		}
		if !containsSubscription(metered, in.SubscriptionId) {
			return domain.MeterEvent{}, lib.NewCustomError(lib.BadRequestError, "subscription does not carry a metered price for this metric", nil)
		}
	}

	value := decimal.Zero
	if metric.CarryOver {
		// Carry-over (stock) meter: the event is either an add/remove for one
		// identity, or a level report (a numeric total). See
		// docs/internal/billing-model/stock-billing-architecture-impact.md §4.
		if op, hasOp := in.Metadata[domain.UsageOperationKey]; hasOp {
			if op != domain.UsageOperationAdd && op != domain.UsageOperationRemove {
				return domain.MeterEvent{}, lib.NewCustomError(lib.BadRequestError, `metadata.operation must be "add" or "remove"`, nil)
			}
			if in.Metadata[metric.FieldName] == "" {
				return domain.MeterEvent{}, lib.NewCustomError(lib.BadRequestError, "metadata is missing the identity field "+metric.FieldName, nil)
			}
		} else {
			raw, ok := in.Metadata[metric.FieldName]
			if !ok {
				return domain.MeterEvent{}, lib.NewCustomError(lib.BadRequestError, "metadata needs either an operation with the identity field "+metric.FieldName+", or a numeric level under "+metric.FieldName, nil)
			}
			v, perr := decimal.NewFromString(raw)
			if perr != nil {
				return domain.MeterEvent{}, lib.NewCustomError(lib.BadRequestError, "metric field "+metric.FieldName+" is not numeric", perr)
			}
			value = v
		}
	} else if metric.Aggregation != domain.AggregationCount && metric.Aggregation != domain.AggregationUniqueCount {
		raw, ok := in.Metadata[metric.FieldName]
		if !ok {
			return domain.MeterEvent{}, lib.NewCustomError(lib.BadRequestError, "metadata is missing the metric field "+metric.FieldName, nil)
		}
		v, perr := decimal.NewFromString(raw)
		if perr != nil {
			return domain.MeterEvent{}, lib.NewCustomError(lib.BadRequestError, "metric field "+metric.FieldName+" is not numeric", perr)
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

	return event, nil
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

	if metric.CarryOver {
		units, err := s.aggregateCarryOver(ctx, metric, q)
		if err != nil {
			return decimal.Zero, err
		}
		return applyRounding(metric, units), nil
	}

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
		// Forbidden at meter creation; defensive guard for any pre-existing row.
		return decimal.Zero, errors.New("weighted_sum requires a carry-over meter")
	default:
		return decimal.Zero, errors.New("unknown aggregation type: " + string(metric.Aggregation))
	}
	if err != nil {
		return decimal.Zero, err
	}
	return applyRounding(metric, units), nil
}

// aggregateCarryOver computes a carry-over (stock) meter's quantity: fetch the
// full event history up to the period end, rebuild the standing level, apply the
// aggregation over [q.From, q.To). Add/remove events become per-identity
// intervals; with no operation events the reported values are the level. See
// docs/internal/billing-model/stock-billing-architecture-impact.md §3/§5.
func (s *UsageService) aggregateCarryOver(ctx context.Context, metric domain.BillableMetric, q port.UsageQuery) (decimal.Decimal, error) {
	hq := q
	hq.From = time.Time{} // events before the period determine the level at its start
	events, err := s.eventStore.ListHistory(ctx, hq)
	if err != nil {
		return decimal.Zero, err
	}

	if domain.HasOperations(events) {
		intervals := domain.ReconstructIntervals(events, metric.FieldName)
		switch metric.Aggregation {
		case domain.AggregationLatest:
			return decimal.NewFromInt(domain.CountStandingAtEnd(intervals, q.To)), nil
		case domain.AggregationMax:
			return decimal.NewFromInt(domain.CountPeakConcurrent(intervals, q.From, q.To)), nil
		case domain.AggregationUniqueCount:
			return decimal.NewFromInt(domain.CountDistinctActive(intervals, q.From, q.To)), nil
		case domain.AggregationWeightedSum:
			return domain.WeightIntervals(intervals, q.From, q.To, q.ProrateOnIncrease, q.CreditOnDecrease), nil
		}
		return decimal.Zero, errors.New("aggregation not supported for carry-over meters: " + string(metric.Aggregation))
	}

	switch metric.Aggregation {
	case domain.AggregationLatest:
		return domain.LastReportedLevel(events), nil
	case domain.AggregationMax:
		return domain.PeakReportedLevel(events, q.From, q.To), nil
	case domain.AggregationUniqueCount:
		return decimal.Zero, nil // level reports carry no identities
	case domain.AggregationWeightedSum:
		return domain.WeightReportedLevels(events, q.From, q.To), nil
	}
	return decimal.Zero, errors.New("aggregation not supported for carry-over meters: " + string(metric.Aggregation))
}

// usageQueryFor resolves the meter from the price and builds the scoped UsageQuery for
// [from, to): customer (incl. the merchant's external id, §8), subscription attribution
// with the earliest-metered-sub catch-all for unattributed usage (§10), and the price's
// filter (one slice of the meter, or the default/catch-all charge). Shared by the scalar
// and grouped read paths.
func (s *UsageService) usageQueryFor(ctx context.Context, sub domain.Subscription, price domain.Price, from, to time.Time) (domain.BillableMetric, port.UsageQuery, error) {
	metric, err := s.meterRepository.FindById(ctx, sub.OrgId, price.BillableMetricId)
	if err != nil {
		return domain.BillableMetric{}, port.UsageQuery{}, err
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
		return domain.BillableMetric{}, port.UsageQuery{}, cerr
	}

	// The earliest metered sub for (customer, meter) is the catch-all for
	// unattributed usage.
	metered, err := s.subscriptionRepository.FindActiveMeteredForMeter(ctx, sub.OrgId, sub.CustomerId, metric.Id)
	if err != nil {
		return domain.BillableMetric{}, port.UsageQuery{}, err
	}
	q.IncludeUnattributed = len(metered) > 0 && metered[0].Id == sub.Id

	// The proration switches are the price's; the carry-over read needs them.
	q.ProrateOnIncrease = price.ProrateOnIncrease
	q.CreditOnDecrease = price.CreditOnDecrease

	applyPriceFilter(&q, metric, price)
	return metric, q, nil
}

// applyPriceFilter scopes a query to the slice of the meter a metered price bills. A
// price with no FilterField bills the whole meter; the default/catch-all charge
// (FilterField set, no value) excludes the field's explicitly-priced values.
func applyPriceFilter(q *port.UsageQuery, metric domain.BillableMetric, price domain.Price) {
	if price.FilterField == "" {
		return
	}
	q.FilterField = price.FilterField
	if price.IsDefaultFilter() {
		q.FilterExclude = metric.FilterValues(price.FilterField)
	} else {
		q.FilterValue = price.FilterValue
	}
}

// UsageForSubscription aggregates a metered subscription's usage for [from, to) into a
// single quantity, honouring the price's filter. Used by the current-period usage read;
// the invoice builder uses MeteredUsageForSubscription so it can split grouped charges.
func (s *UsageService) UsageForSubscription(ctx context.Context, sub domain.Subscription, price domain.Price, from, to time.Time) (decimal.Decimal, error) {
	metric, q, err := s.usageQueryFor(ctx, sub, price, from, to)
	if err != nil {
		return decimal.Zero, err
	}
	return s.AggregateForPeriod(ctx, metric, q)
}

// MeteredUsage is a metered price's usage for a period: a single scalar quantity when
// the meter has no group dimension, or one segment per group value when it does
// (Grouped non-nil). The caller emits one invoice line for Units, or one per segment.
type MeteredUsage struct {
	Units   decimal.Decimal
	Grouped []port.GroupedUsage
}

// MeteredUsageForSubscription returns a metered price's usage for [from, to): the
// filtered scalar, or — when the meter declares a GroupBy dimension — one rounded
// quantity per discovered group value (all at the price's single rate). v1 honours one
// group dimension; a meter with more errors.
func (s *UsageService) MeteredUsageForSubscription(ctx context.Context, sub domain.Subscription, price domain.Price, from, to time.Time) (MeteredUsage, error) {
	metric, q, err := s.usageQueryFor(ctx, sub, price, from, to)
	if err != nil {
		return MeteredUsage{}, err
	}
	if len(metric.GroupBy) == 0 {
		units, aerr := s.AggregateForPeriod(ctx, metric, q)
		return MeteredUsage{Units: units}, aerr
	}
	if len(metric.GroupBy) > 1 {
		return MeteredUsage{}, errors.New("multi-dimension grouping not implemented")
	}
	q.FieldName = metric.FieldName
	q.MetricCode = metric.Code
	groups, gerr := s.eventStore.AggregateGrouped(ctx, q, metric.Aggregation, metric.GroupBy[0])
	if gerr != nil {
		return MeteredUsage{}, gerr
	}
	for i := range groups {
		groups[i].Quantity = applyRounding(metric, groups[i].Quantity)
	}
	return MeteredUsage{Grouped: groups}, nil
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
