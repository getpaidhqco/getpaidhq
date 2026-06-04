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
	var sub domain.Subscription
	err := dbFromCtx(ctx, r.db).
		Scopes(OrgScope(orgId)).
		Where("id = ?", id).
		Preload("Customer").
		First(&sub).Error
	return sub, translateErr(err)
}

// FindByIdForUpdate is the row-locking variant of FindById. MUST be
// called inside a transaction (TxManager.RunInTx); outside a tx the
// lock is acquired and immediately released, which defeats the
// purpose. The Postgres dialect emits SELECT ... FOR UPDATE.
func (r *SubscriptionRepo) FindByIdForUpdate(ctx context.Context, orgId string, id string) (domain.Subscription, error) {
	var sub domain.Subscription
	err := dbFromCtx(ctx, r.db).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Scopes(OrgScope(orgId)).
		Where("id = ?", id).
		Preload("Customer").
		First(&sub).Error
	return sub, translateErr(err)
}

func (r *SubscriptionRepo) Create(ctx context.Context, entity domain.Subscription) (domain.Subscription, error) {
	entity.Metadata = emptyIfNil(entity.Metadata)
	err := dbFromCtx(ctx, r.db).Create(&entity).Error
	if err != nil {
		return domain.Subscription{}, err
	}
	return r.FindById(ctx, entity.OrgId, entity.Id)
}

func (r *SubscriptionRepo) Update(ctx context.Context, entity domain.Subscription) (domain.Subscription, error) {
	err := dbFromCtx(ctx, r.db).Save(&entity).Error
	if err != nil {
		return domain.Subscription{}, err
	}
	return r.FindById(ctx, entity.OrgId, entity.Id)
}

func (r *SubscriptionRepo) FindByOrderId(ctx context.Context, orgId string, orderId string) ([]domain.Subscription, error) {
	var subs []domain.Subscription
	err := dbFromCtx(ctx, r.db).
		Scopes(OrgScope(orgId)).
		Where("order_id = ?", orderId).
		Find(&subs).Error
	return subs, err
}

func (r *SubscriptionRepo) Find(ctx context.Context, orgId string, p domain.Pagination) ([]domain.Subscription, int, error) {
	var subs []domain.Subscription
	var count int64
	err := dbFromCtx(ctx, r.db).Model(&domain.Subscription{}).
		Scopes(OrgScope(orgId)).
		Count(&count).Error
	if err != nil {
		return nil, 0, err
	}
	err = dbFromCtx(ctx, r.db).
		Scopes(OrgScope(orgId), Paginate(p)).
		Preload("Customer").
		Find(&subs).Error
	return subs, int(count), err
}

// FindDueForBilling selects subscriptions due for a charge now. Keep the status/
// date rule below in sync with domain.Subscription.IsDueForBilling — that Go method
// is the per-subscription mirror of this SQL (used by the Hatchet activation spawn),
// and the two must agree on what "due" means.
func (r *SubscriptionRepo) FindDueForBilling(ctx context.Context, orgId string, now time.Time) ([]domain.Subscription, error) {
	var subs []domain.Subscription
	// Unset date columns are NULL (serializer:nulltime maps zero time → NULL),
	// and `col <= now` is already false for NULL, so unset rows are auto-excluded.
	err := dbFromCtx(ctx, r.db).
		Scopes(OrgScope(orgId)).
		Where(
			r.db.Where("status = ? AND renews_at <= ?", domain.SubscriptionStatusActive, now).
				Or("status = ? AND next_retry <= ?", domain.SubscriptionStatusPastDue, now).
				Or("status = ? AND trial_ends_at <= ?", domain.SubscriptionStatusTrial, now),
		).
		Find(&subs).Error
	return subs, err
}

func (r *SubscriptionRepo) FindUpcomingRenewals(ctx context.Context, orgId string, now time.Time, within time.Duration) ([]domain.Subscription, error) {
	var subs []domain.Subscription
	err := dbFromCtx(ctx, r.db).
		Scopes(OrgScope(orgId)).
		Where("status = ? AND renews_at > ? AND renews_at <= ?",
			domain.SubscriptionStatusActive, now, now.Add(within)).
		Find(&subs).Error
	return subs, err
}
