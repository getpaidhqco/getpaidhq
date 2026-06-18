package postgresgorm

import (
	"context"
	"errors"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"

	"gorm.io/gorm"
)

type DunningRepo struct {
	db *gorm.DB
}

func NewDunningRepo(db *gorm.DB) port.DunningRepository {
	return &DunningRepo{db: db}
}

// ---- Campaigns ----

func (r *DunningRepo) CreateCampaign(ctx context.Context, c domain.DunningCampaign) (domain.DunningCampaign, error) {
	row := dunningCampaignRowFromDomain(c)
	if err := dbFromCtx(ctx, r.db).Create(&row).Error; err != nil {
		return domain.DunningCampaign{}, err
	}
	return r.FindCampaignById(ctx, c.OrgId, c.Id)
}

func (r *DunningRepo) FindCampaignById(ctx context.Context, orgId, id string) (domain.DunningCampaign, error) {
	var row dunningCampaignRow
	err := dbFromCtx(ctx, r.db).
		Scopes(OrgScope(orgId)).
		Where("id = ?", id).
		First(&row).Error
	if err != nil {
		return domain.DunningCampaign{}, translateErr(err)
	}
	return row.toDomain(), nil
}

func (r *DunningRepo) FindCampaigns(ctx context.Context, orgId string, p domain.Pagination) ([]domain.DunningCampaign, int, error) {
	var rows []dunningCampaignRow
	var count int64
	if err := dbFromCtx(ctx, r.db).Model(&dunningCampaignRow{}).Scopes(OrgScope(orgId)).Count(&count).Error; err != nil {
		return nil, 0, err
	}
	if err := dbFromCtx(ctx, r.db).Scopes(OrgScope(orgId), Paginate(p)).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return dunningCampaignRowsToDomain(rows), int(count), nil
}

func (r *DunningRepo) FindCampaignsBySubscriptionId(ctx context.Context, orgId, subscriptionId string, p domain.Pagination) ([]domain.DunningCampaign, int, error) {
	var rows []dunningCampaignRow
	var count int64
	q := dbFromCtx(ctx, r.db).Model(&dunningCampaignRow{}).Scopes(OrgScope(orgId)).Where("subscription_id = ?", subscriptionId)
	if err := q.Count(&count).Error; err != nil {
		return nil, 0, err
	}
	if err := dbFromCtx(ctx, r.db).Scopes(OrgScope(orgId), Paginate(p)).Where("subscription_id = ?", subscriptionId).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return dunningCampaignRowsToDomain(rows), int(count), nil
}

func (r *DunningRepo) FindCampaignsByCustomerId(ctx context.Context, orgId, customerId string, p domain.Pagination) ([]domain.DunningCampaign, int, error) {
	var rows []dunningCampaignRow
	var count int64
	q := dbFromCtx(ctx, r.db).Model(&dunningCampaignRow{}).Scopes(OrgScope(orgId)).Where("customer_id = ?", customerId)
	if err := q.Count(&count).Error; err != nil {
		return nil, 0, err
	}
	if err := dbFromCtx(ctx, r.db).Scopes(OrgScope(orgId), Paginate(p)).Where("customer_id = ?", customerId).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return dunningCampaignRowsToDomain(rows), int(count), nil
}

func (r *DunningRepo) FindActiveCampaignForSubscription(ctx context.Context, orgId, subscriptionId string) (domain.DunningCampaign, error) {
	var row dunningCampaignRow
	err := dbFromCtx(ctx, r.db).
		Scopes(OrgScope(orgId)).
		Where("subscription_id = ? AND status IN ?", subscriptionId, []domain.DunningStatus{domain.DunningStatusActive, domain.DunningStatusPaused}).
		Order("created_at DESC").
		First(&row).Error
	if err != nil {
		return domain.DunningCampaign{}, translateErr(err)
	}
	return row.toDomain(), nil
}

func (r *DunningRepo) UpdateCampaign(ctx context.Context, c domain.DunningCampaign) (domain.DunningCampaign, error) {
	row := dunningCampaignRowFromDomain(c)
	if err := dbFromCtx(ctx, r.db).Save(&row).Error; err != nil {
		return domain.DunningCampaign{}, err
	}
	return r.FindCampaignById(ctx, c.OrgId, c.Id)
}

// ---- Attempts ----

func (r *DunningRepo) CreateAttempt(ctx context.Context, a domain.DunningAttempt) (domain.DunningAttempt, error) {
	row := dunningAttemptRowFromDomain(a)
	if err := dbFromCtx(ctx, r.db).Create(&row).Error; err != nil {
		return domain.DunningAttempt{}, err
	}
	return r.FindAttemptById(ctx, a.OrgId, a.Id)
}

func (r *DunningRepo) FindAttemptById(ctx context.Context, orgId, id string) (domain.DunningAttempt, error) {
	var row dunningAttemptRow
	err := dbFromCtx(ctx, r.db).Scopes(OrgScope(orgId)).Where("id = ?", id).First(&row).Error
	if err != nil {
		return domain.DunningAttempt{}, translateErr(err)
	}
	return row.toDomain(), nil
}

func (r *DunningRepo) FindAttemptsByCampaignId(ctx context.Context, orgId, campaignId string, p domain.Pagination) ([]domain.DunningAttempt, int, error) {
	var rows []dunningAttemptRow
	var count int64
	q := dbFromCtx(ctx, r.db).Model(&dunningAttemptRow{}).Scopes(OrgScope(orgId)).Where("dunning_campaign_id = ?", campaignId)
	if err := q.Count(&count).Error; err != nil {
		return nil, 0, err
	}
	if err := dbFromCtx(ctx, r.db).Scopes(OrgScope(orgId), Paginate(p)).Where("dunning_campaign_id = ?", campaignId).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return dunningAttemptRowsToDomain(rows), int(count), nil
}

// ---- Communications ----

func (r *DunningRepo) CreateCommunication(ctx context.Context, c domain.DunningCommunication) (domain.DunningCommunication, error) {
	row := dunningCommunicationRowFromDomain(c)
	if err := dbFromCtx(ctx, r.db).Create(&row).Error; err != nil {
		return domain.DunningCommunication{}, err
	}
	return r.FindCommunicationById(ctx, c.OrgId, c.Id)
}

func (r *DunningRepo) FindCommunicationById(ctx context.Context, orgId, id string) (domain.DunningCommunication, error) {
	var row dunningCommunicationRow
	err := dbFromCtx(ctx, r.db).Scopes(OrgScope(orgId)).Where("id = ?", id).First(&row).Error
	if err != nil {
		return domain.DunningCommunication{}, translateErr(err)
	}
	return row.toDomain(), nil
}

func (r *DunningRepo) FindCommunicationsByCampaignId(ctx context.Context, orgId, campaignId string, p domain.Pagination) ([]domain.DunningCommunication, int, error) {
	var rows []dunningCommunicationRow
	var count int64
	q := dbFromCtx(ctx, r.db).Model(&dunningCommunicationRow{}).Scopes(OrgScope(orgId)).Where("dunning_campaign_id = ?", campaignId)
	if err := q.Count(&count).Error; err != nil {
		return nil, 0, err
	}
	if err := dbFromCtx(ctx, r.db).Scopes(OrgScope(orgId), Paginate(p)).Where("dunning_campaign_id = ?", campaignId).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return dunningCommunicationRowsToDomain(rows), int(count), nil
}

func (r *DunningRepo) UpdateCommunication(ctx context.Context, c domain.DunningCommunication) (domain.DunningCommunication, error) {
	row := dunningCommunicationRowFromDomain(c)
	if err := dbFromCtx(ctx, r.db).Save(&row).Error; err != nil {
		return domain.DunningCommunication{}, err
	}
	return r.FindCommunicationById(ctx, c.OrgId, c.Id)
}

// ---- Tokens ----

func (r *DunningRepo) CreateToken(ctx context.Context, t domain.PaymentUpdateToken) (domain.PaymentUpdateToken, error) {
	row := paymentUpdateTokenRowFromDomain(t)
	if err := dbFromCtx(ctx, r.db).Create(&row).Error; err != nil {
		return domain.PaymentUpdateToken{}, err
	}
	return r.FindTokenById(ctx, t.OrgId, t.TokenId)
}

func (r *DunningRepo) FindTokenById(ctx context.Context, orgId, tokenId string) (domain.PaymentUpdateToken, error) {
	var row paymentUpdateTokenRow
	err := dbFromCtx(ctx, r.db).Scopes(OrgScope(orgId)).Where("token_id = ?", tokenId).First(&row).Error
	if err != nil {
		return domain.PaymentUpdateToken{}, translateErr(err)
	}
	return row.toDomain(), nil
}

func (r *DunningRepo) FindTokensBySubscriptionId(ctx context.Context, orgId, subscriptionId string, p domain.Pagination) ([]domain.PaymentUpdateToken, int, error) {
	var rows []paymentUpdateTokenRow
	var count int64
	q := dbFromCtx(ctx, r.db).Model(&paymentUpdateTokenRow{}).Scopes(OrgScope(orgId)).Where("subscription_id = ?", subscriptionId)
	if err := q.Count(&count).Error; err != nil {
		return nil, 0, err
	}
	if err := dbFromCtx(ctx, r.db).Scopes(OrgScope(orgId), Paginate(p)).Where("subscription_id = ?", subscriptionId).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return paymentUpdateTokenRowsToDomain(rows), int(count), nil
}

func (r *DunningRepo) FindTokensByCampaignId(ctx context.Context, orgId, campaignId string, p domain.Pagination) ([]domain.PaymentUpdateToken, int, error) {
	var rows []paymentUpdateTokenRow
	var count int64
	q := dbFromCtx(ctx, r.db).Model(&paymentUpdateTokenRow{}).Scopes(OrgScope(orgId)).Where("dunning_campaign_id = ?", campaignId)
	if err := q.Count(&count).Error; err != nil {
		return nil, 0, err
	}
	if err := dbFromCtx(ctx, r.db).Scopes(OrgScope(orgId), Paginate(p)).Where("dunning_campaign_id = ?", campaignId).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return paymentUpdateTokenRowsToDomain(rows), int(count), nil
}

func (r *DunningRepo) UpdateToken(ctx context.Context, t domain.PaymentUpdateToken) (domain.PaymentUpdateToken, error) {
	row := paymentUpdateTokenRowFromDomain(t)
	if err := dbFromCtx(ctx, r.db).Save(&row).Error; err != nil {
		return domain.PaymentUpdateToken{}, err
	}
	return r.FindTokenById(ctx, t.OrgId, t.TokenId)
}

// ---- Configurations ----

func (r *DunningRepo) CreateConfiguration(ctx context.Context, c domain.DunningConfiguration) (domain.DunningConfiguration, error) {
	row := dunningConfigurationRowFromDomain(c)
	if err := dbFromCtx(ctx, r.db).Create(&row).Error; err != nil {
		return domain.DunningConfiguration{}, err
	}
	return r.FindConfigurationById(ctx, c.OrgId, c.Id)
}

func (r *DunningRepo) FindConfigurationById(ctx context.Context, orgId, id string) (domain.DunningConfiguration, error) {
	var row dunningConfigurationRow
	err := dbFromCtx(ctx, r.db).Scopes(OrgScope(orgId)).Where("id = ?", id).First(&row).Error
	if err != nil {
		return domain.DunningConfiguration{}, translateErr(err)
	}
	return row.toDomain(), nil
}

func (r *DunningRepo) FindConfigurations(ctx context.Context, orgId string, p domain.Pagination) ([]domain.DunningConfiguration, int, error) {
	var rows []dunningConfigurationRow
	var count int64
	if err := dbFromCtx(ctx, r.db).Model(&dunningConfigurationRow{}).Scopes(OrgScope(orgId)).Count(&count).Error; err != nil {
		return nil, 0, err
	}
	if err := dbFromCtx(ctx, r.db).Scopes(OrgScope(orgId), Paginate(p)).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return dunningConfigurationRowsToDomain(rows), int(count), nil
}

func (r *DunningRepo) FindConfigurationsByPriority(ctx context.Context, orgId string) ([]domain.DunningConfiguration, error) {
	var rows []dunningConfigurationRow
	err := dbFromCtx(ctx, r.db).
		Scopes(OrgScope(orgId)).
		Where("status = ?", domain.ConfigStatusActive).
		Order("priority DESC").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	return dunningConfigurationRowsToDomain(rows), nil
}

func (r *DunningRepo) UpdateConfiguration(ctx context.Context, c domain.DunningConfiguration) (domain.DunningConfiguration, error) {
	row := dunningConfigurationRowFromDomain(c)
	if err := dbFromCtx(ctx, r.db).Save(&row).Error; err != nil {
		return domain.DunningConfiguration{}, err
	}
	return r.FindConfigurationById(ctx, c.OrgId, c.Id)
}

// ---- Customer history ----

func (r *DunningRepo) GetCustomerDunningHistory(ctx context.Context, orgId, customerId string) (domain.CustomerDunningHistory, error) {
	var row customerDunningHistoryRow
	err := dbFromCtx(ctx, r.db).Scopes(OrgScope(orgId)).Where("customer_id = ?", customerId).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// Synthesize a zero history rather than propagate "not found" —
		// callers treat the absence of any dunning row as "clean".
		return domain.CustomerDunningHistory{
			OrgId:      orgId,
			CustomerId: customerId,
		}, nil
	}
	if err != nil {
		return domain.CustomerDunningHistory{}, translateErr(err)
	}
	return row.toDomain(), nil
}

func (r *DunningRepo) UpsertCustomerDunningHistory(ctx context.Context, h domain.CustomerDunningHistory) (domain.CustomerDunningHistory, error) {
	row := customerDunningHistoryRowFromDomain(h)
	q := dbFromCtx(ctx, r.db)
	// most_responsive_channel is a nullable enum; omit when empty so that an empty
	// string is not sent to postgres instead of NULL.
	if h.MostResponsiveChannel == "" {
		q = q.Omit("most_responsive_channel")
	}
	if err := q.Save(&row).Error; err != nil {
		return domain.CustomerDunningHistory{}, err
	}
	return r.GetCustomerDunningHistory(ctx, h.OrgId, h.CustomerId)
}
