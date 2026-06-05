package postgres

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
)

type SubscriptionRepo struct {
	db *gorm.DB
}

func NewSubscriptionRepo(db *gorm.DB) port.SubscriptionRepository {
	return &SubscriptionRepo{db: db}
}

func (r *SubscriptionRepo) FindById(ctx context.Context, orgId string, id string) (domain.Subscription, error) {
	var row subscriptionRow
	err := dbFromCtx(ctx, r.db).
		Scopes(OrgScope(orgId)).
		Where("id = ?", id).
		First(&row).Error
	if err != nil {
		return domain.Subscription{}, translateErr(err)
	}
	return row.toDomain(), nil
}

// FindByIdForUpdate is the row-locking variant of FindById. MUST be
// called inside a transaction (TxManager.RunInTx); outside a tx the
// lock is acquired and immediately released, which defeats the
// purpose. The Postgres dialect emits SELECT ... FOR UPDATE.
func (r *SubscriptionRepo) FindByIdForUpdate(ctx context.Context, orgId string, id string) (domain.Subscription, error) {
	var row subscriptionRow
	err := dbFromCtx(ctx, r.db).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Scopes(OrgScope(orgId)).
		Where("id = ?", id).
		First(&row).Error
	if err != nil {
		return domain.Subscription{}, translateErr(err)
	}
	return row.toDomain(), nil
}

func (r *SubscriptionRepo) Create(ctx context.Context, entity domain.Subscription) (domain.Subscription, error) {
	entity.Metadata = emptyIfNil(entity.Metadata)
	row := subscriptionRowFromDomain(entity)
	if err := dbFromCtx(ctx, r.db).Omit("Customer", "OrderItem").Create(&row).Error; err != nil {
		return domain.Subscription{}, err
	}
	return r.FindById(ctx, entity.OrgId, entity.Id)
}

func (r *SubscriptionRepo) Update(ctx context.Context, entity domain.Subscription) (domain.Subscription, error) {
	row := subscriptionRowFromDomain(entity)
	if err := dbFromCtx(ctx, r.db).Omit("Customer", "OrderItem").Save(&row).Error; err != nil {
		return domain.Subscription{}, err
	}
	return r.FindById(ctx, entity.OrgId, entity.Id)
}

func (r *SubscriptionRepo) FindByOrderId(ctx context.Context, orgId string, orderId string) ([]domain.Subscription, error) {
	var rows []subscriptionRow
	err := dbFromCtx(ctx, r.db).
		Scopes(OrgScope(orgId)).
		Where("order_id = ?", orderId).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	return subscriptionRowsToDomain(rows), nil
}

func (r *SubscriptionRepo) FindActiveMeteredForMeter(ctx context.Context, orgId, customerId, billableMetricId string) ([]domain.Subscription, error) {
	var rows []subscriptionRow
	// A subscription is "metered for M" when ANY item in its order carries a metered
	// price for meter M (the plan subscription bills its order's usage items — the
	// metered items don't create their own subscriptions). DISTINCT collapses orders
	// that carry several metered items for the same meter.
	err := dbFromCtx(ctx, r.db).
		Model(&subscriptionRow{}).
		Distinct("subscriptions.*").
		Joins("JOIN order_items oi ON oi.org_id = subscriptions.org_id AND oi.order_id = subscriptions.order_id").
		Joins("JOIN prices p ON p.org_id = oi.org_id AND p.id = oi.price_id").
		Where("subscriptions.org_id = ? AND subscriptions.customer_id = ?", orgId, customerId).
		Where("p.category = ? AND p.billable_metric_id = ?", domain.PriceCategoryMetered, billableMetricId).
		Where("subscriptions.status IN ?", []string{
			string(domain.SubscriptionStatusActive),
			string(domain.SubscriptionStatusTrial),
			string(domain.SubscriptionStatusPastDue),
		}).
		Order("subscriptions.start_date ASC, subscriptions.created_at ASC").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	return subscriptionRowsToDomain(rows), nil
}

func (r *SubscriptionRepo) Find(ctx context.Context, orgId string, p domain.Pagination) ([]domain.Subscription, int, error) {
	var rows []subscriptionRow
	var count int64
	if err := dbFromCtx(ctx, r.db).Model(&subscriptionRow{}).
		Scopes(OrgScope(orgId)).
		Count(&count).Error; err != nil {
		return nil, 0, err
	}
	if err := dbFromCtx(ctx, r.db).
		Scopes(OrgScope(orgId), Paginate(p)).
		Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return subscriptionRowsToDomain(rows), int(count), nil
}

// FindDueForBilling selects subscriptions due for a charge now. Keep the status/
// date rule below in sync with domain.Subscription.IsDueForBilling — that Go method
// is the per-subscription mirror of this SQL (used by the Hatchet activation spawn),
// and the two must agree on what "due" means.
func (r *SubscriptionRepo) FindDueForBilling(ctx context.Context, orgId string, now time.Time) ([]domain.Subscription, error) {
	var rows []subscriptionRow
	// Unset date columns are NULL (serializer:nulltime maps zero time → NULL),
	// and `col <= now` is already false for NULL, so unset rows are auto-excluded.
	err := dbFromCtx(ctx, r.db).
		Scopes(OrgScope(orgId)).
		Where(
			r.db.Where("status = ? AND renews_at <= ?", domain.SubscriptionStatusActive, now).
				Or("status = ? AND next_retry <= ?", domain.SubscriptionStatusPastDue, now).
				Or("status = ? AND trial_ends_at <= ?", domain.SubscriptionStatusTrial, now),
		).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	return subscriptionRowsToDomain(rows), nil
}

func (r *SubscriptionRepo) FindUpcomingRenewals(ctx context.Context, orgId string, now time.Time, within time.Duration) ([]domain.Subscription, error) {
	var rows []subscriptionRow
	err := dbFromCtx(ctx, r.db).
		Scopes(OrgScope(orgId)).
		Where("status = ? AND renews_at > ? AND renews_at <= ?",
			domain.SubscriptionStatusActive, now, now.Add(within)).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	return subscriptionRowsToDomain(rows), nil
}
