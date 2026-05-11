package postgres

import (
	"context"
	"errors"

	"gorm.io/gorm"
	"payloop/internal/core/domain"
	"payloop/internal/core/port"
)

type DunningRepo struct {
	db *gorm.DB
}

func NewDunningRepo(db *gorm.DB) port.DunningRepository {
	return &DunningRepo{db: db}
}

// ---- Campaigns ----

func (r *DunningRepo) CreateCampaign(ctx context.Context, c domain.DunningCampaign) (domain.DunningCampaign, error) {
	if err := r.db.WithContext(ctx).Create(&c).Error; err != nil {
		return domain.DunningCampaign{}, err
	}
	return r.FindCampaignById(ctx, c.OrgId, c.Id)
}

func (r *DunningRepo) FindCampaignById(ctx context.Context, orgId, id string) (domain.DunningCampaign, error) {
	var c domain.DunningCampaign
	err := r.db.WithContext(ctx).
		Scopes(OrgScope(orgId)).
		Where("id = ?", id).
		First(&c).Error
	return c, err
}

func (r *DunningRepo) FindCampaigns(ctx context.Context, orgId string, p domain.Pagination) ([]domain.DunningCampaign, int, error) {
	var cs []domain.DunningCampaign
	var count int64
	if err := r.db.WithContext(ctx).Model(&domain.DunningCampaign{}).Scopes(OrgScope(orgId)).Count(&count).Error; err != nil {
		return nil, 0, err
	}
	if err := r.db.WithContext(ctx).Scopes(OrgScope(orgId), Paginate(p)).Find(&cs).Error; err != nil {
		return nil, 0, err
	}
	return cs, int(count), nil
}

func (r *DunningRepo) FindCampaignsBySubscriptionId(ctx context.Context, orgId, subscriptionId string, p domain.Pagination) ([]domain.DunningCampaign, int, error) {
	var cs []domain.DunningCampaign
	var count int64
	q := r.db.WithContext(ctx).Model(&domain.DunningCampaign{}).Scopes(OrgScope(orgId)).Where("subscription_id = ?", subscriptionId)
	if err := q.Count(&count).Error; err != nil {
		return nil, 0, err
	}
	if err := r.db.WithContext(ctx).Scopes(OrgScope(orgId), Paginate(p)).Where("subscription_id = ?", subscriptionId).Find(&cs).Error; err != nil {
		return nil, 0, err
	}
	return cs, int(count), nil
}

func (r *DunningRepo) FindCampaignsByCustomerId(ctx context.Context, orgId, customerId string, p domain.Pagination) ([]domain.DunningCampaign, int, error) {
	var cs []domain.DunningCampaign
	var count int64
	q := r.db.WithContext(ctx).Model(&domain.DunningCampaign{}).Scopes(OrgScope(orgId)).Where("customer_id = ?", customerId)
	if err := q.Count(&count).Error; err != nil {
		return nil, 0, err
	}
	if err := r.db.WithContext(ctx).Scopes(OrgScope(orgId), Paginate(p)).Where("customer_id = ?", customerId).Find(&cs).Error; err != nil {
		return nil, 0, err
	}
	return cs, int(count), nil
}

func (r *DunningRepo) FindActiveCampaignForSubscription(ctx context.Context, orgId, subscriptionId string) (domain.DunningCampaign, error) {
	var c domain.DunningCampaign
	err := r.db.WithContext(ctx).
		Scopes(OrgScope(orgId)).
		Where("subscription_id = ? AND status IN ?", subscriptionId, []domain.DunningStatus{domain.DunningStatusActive, domain.DunningStatusPaused}).
		Order("created_at DESC").
		First(&c).Error
	return c, err
}

func (r *DunningRepo) UpdateCampaign(ctx context.Context, c domain.DunningCampaign) (domain.DunningCampaign, error) {
	if err := r.db.WithContext(ctx).Save(&c).Error; err != nil {
		return domain.DunningCampaign{}, err
	}
	return r.FindCampaignById(ctx, c.OrgId, c.Id)
}

// ---- Attempts ----

func (r *DunningRepo) CreateAttempt(ctx context.Context, a domain.DunningAttempt) (domain.DunningAttempt, error) {
	if err := r.db.WithContext(ctx).Create(&a).Error; err != nil {
		return domain.DunningAttempt{}, err
	}
	return r.FindAttemptById(ctx, a.OrgId, a.Id)
}

func (r *DunningRepo) FindAttemptById(ctx context.Context, orgId, id string) (domain.DunningAttempt, error) {
	var a domain.DunningAttempt
	err := r.db.WithContext(ctx).Scopes(OrgScope(orgId)).Where("id = ?", id).First(&a).Error
	return a, err
}

func (r *DunningRepo) FindAttemptsByCampaignId(ctx context.Context, orgId, campaignId string, p domain.Pagination) ([]domain.DunningAttempt, int, error) {
	var as []domain.DunningAttempt
	var count int64
	q := r.db.WithContext(ctx).Model(&domain.DunningAttempt{}).Scopes(OrgScope(orgId)).Where("dunning_campaign_id = ?", campaignId)
	if err := q.Count(&count).Error; err != nil {
		return nil, 0, err
	}
	if err := r.db.WithContext(ctx).Scopes(OrgScope(orgId), Paginate(p)).Where("dunning_campaign_id = ?", campaignId).Find(&as).Error; err != nil {
		return nil, 0, err
	}
	return as, int(count), nil
}

// ---- Communications ----

func (r *DunningRepo) CreateCommunication(ctx context.Context, c domain.DunningCommunication) (domain.DunningCommunication, error) {
	if err := r.db.WithContext(ctx).Create(&c).Error; err != nil {
		return domain.DunningCommunication{}, err
	}
	return r.FindCommunicationById(ctx, c.OrgId, c.Id)
}

func (r *DunningRepo) FindCommunicationById(ctx context.Context, orgId, id string) (domain.DunningCommunication, error) {
	var c domain.DunningCommunication
	err := r.db.WithContext(ctx).Scopes(OrgScope(orgId)).Where("id = ?", id).First(&c).Error
	return c, err
}

func (r *DunningRepo) FindCommunicationsByCampaignId(ctx context.Context, orgId, campaignId string, p domain.Pagination) ([]domain.DunningCommunication, int, error) {
	var cs []domain.DunningCommunication
	var count int64
	q := r.db.WithContext(ctx).Model(&domain.DunningCommunication{}).Scopes(OrgScope(orgId)).Where("dunning_campaign_id = ?", campaignId)
	if err := q.Count(&count).Error; err != nil {
		return nil, 0, err
	}
	if err := r.db.WithContext(ctx).Scopes(OrgScope(orgId), Paginate(p)).Where("dunning_campaign_id = ?", campaignId).Find(&cs).Error; err != nil {
		return nil, 0, err
	}
	return cs, int(count), nil
}

func (r *DunningRepo) UpdateCommunication(ctx context.Context, c domain.DunningCommunication) (domain.DunningCommunication, error) {
	if err := r.db.WithContext(ctx).Save(&c).Error; err != nil {
		return domain.DunningCommunication{}, err
	}
	return r.FindCommunicationById(ctx, c.OrgId, c.Id)
}

// ---- Tokens ----

func (r *DunningRepo) CreateToken(ctx context.Context, t domain.PaymentUpdateToken) (domain.PaymentUpdateToken, error) {
	if err := r.db.WithContext(ctx).Create(&t).Error; err != nil {
		return domain.PaymentUpdateToken{}, err
	}
	return r.FindTokenById(ctx, t.OrgId, t.TokenId)
}

func (r *DunningRepo) FindTokenById(ctx context.Context, orgId, tokenId string) (domain.PaymentUpdateToken, error) {
	var t domain.PaymentUpdateToken
	err := r.db.WithContext(ctx).Scopes(OrgScope(orgId)).Where("token_id = ?", tokenId).First(&t).Error
	return t, err
}

func (r *DunningRepo) FindTokensBySubscriptionId(ctx context.Context, orgId, subscriptionId string, p domain.Pagination) ([]domain.PaymentUpdateToken, int, error) {
	var ts []domain.PaymentUpdateToken
	var count int64
	q := r.db.WithContext(ctx).Model(&domain.PaymentUpdateToken{}).Scopes(OrgScope(orgId)).Where("subscription_id = ?", subscriptionId)
	if err := q.Count(&count).Error; err != nil {
		return nil, 0, err
	}
	if err := r.db.WithContext(ctx).Scopes(OrgScope(orgId), Paginate(p)).Where("subscription_id = ?", subscriptionId).Find(&ts).Error; err != nil {
		return nil, 0, err
	}
	return ts, int(count), nil
}

func (r *DunningRepo) FindTokensByCampaignId(ctx context.Context, orgId, campaignId string, p domain.Pagination) ([]domain.PaymentUpdateToken, int, error) {
	var ts []domain.PaymentUpdateToken
	var count int64
	q := r.db.WithContext(ctx).Model(&domain.PaymentUpdateToken{}).Scopes(OrgScope(orgId)).Where("dunning_campaign_id = ?", campaignId)
	if err := q.Count(&count).Error; err != nil {
		return nil, 0, err
	}
	if err := r.db.WithContext(ctx).Scopes(OrgScope(orgId), Paginate(p)).Where("dunning_campaign_id = ?", campaignId).Find(&ts).Error; err != nil {
		return nil, 0, err
	}
	return ts, int(count), nil
}

func (r *DunningRepo) UpdateToken(ctx context.Context, t domain.PaymentUpdateToken) (domain.PaymentUpdateToken, error) {
	if err := r.db.WithContext(ctx).Save(&t).Error; err != nil {
		return domain.PaymentUpdateToken{}, err
	}
	return r.FindTokenById(ctx, t.OrgId, t.TokenId)
}

// ---- Configurations ----

func (r *DunningRepo) CreateConfiguration(ctx context.Context, c domain.DunningConfiguration) (domain.DunningConfiguration, error) {
	if err := r.db.WithContext(ctx).Create(&c).Error; err != nil {
		return domain.DunningConfiguration{}, err
	}
	return r.FindConfigurationById(ctx, c.OrgId, c.Id)
}

func (r *DunningRepo) FindConfigurationById(ctx context.Context, orgId, id string) (domain.DunningConfiguration, error) {
	var c domain.DunningConfiguration
	err := r.db.WithContext(ctx).Scopes(OrgScope(orgId)).Where("id = ?", id).First(&c).Error
	return c, err
}

func (r *DunningRepo) FindConfigurations(ctx context.Context, orgId string, p domain.Pagination) ([]domain.DunningConfiguration, int, error) {
	var cs []domain.DunningConfiguration
	var count int64
	if err := r.db.WithContext(ctx).Model(&domain.DunningConfiguration{}).Scopes(OrgScope(orgId)).Count(&count).Error; err != nil {
		return nil, 0, err
	}
	if err := r.db.WithContext(ctx).Scopes(OrgScope(orgId), Paginate(p)).Find(&cs).Error; err != nil {
		return nil, 0, err
	}
	return cs, int(count), nil
}

func (r *DunningRepo) FindConfigurationsByPriority(ctx context.Context, orgId string) ([]domain.DunningConfiguration, error) {
	var cs []domain.DunningConfiguration
	err := r.db.WithContext(ctx).
		Scopes(OrgScope(orgId)).
		Where("status = ?", domain.ConfigStatusActive).
		Order("priority DESC").
		Find(&cs).Error
	return cs, err
}

func (r *DunningRepo) UpdateConfiguration(ctx context.Context, c domain.DunningConfiguration) (domain.DunningConfiguration, error) {
	if err := r.db.WithContext(ctx).Save(&c).Error; err != nil {
		return domain.DunningConfiguration{}, err
	}
	return r.FindConfigurationById(ctx, c.OrgId, c.Id)
}

// ---- Customer history ----

func (r *DunningRepo) GetCustomerDunningHistory(ctx context.Context, orgId, customerId string) (domain.CustomerDunningHistory, error) {
	var h domain.CustomerDunningHistory
	err := r.db.WithContext(ctx).Scopes(OrgScope(orgId)).Where("customer_id = ?", customerId).First(&h).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.CustomerDunningHistory{
			OrgId:      orgId,
			CustomerId: customerId,
		}, nil
	}
	return h, err
}

func (r *DunningRepo) UpsertCustomerDunningHistory(ctx context.Context, h domain.CustomerDunningHistory) (domain.CustomerDunningHistory, error) {
	if err := r.db.WithContext(ctx).Save(&h).Error; err != nil {
		return domain.CustomerDunningHistory{}, err
	}
	return r.GetCustomerDunningHistory(ctx, h.OrgId, h.CustomerId)
}
