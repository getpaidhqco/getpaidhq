package postgres

import (
	"context"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"

	"gorm.io/gorm"
)

type CustomerRepo struct {
	db *gorm.DB
}

func NewCustomerRepo(db *gorm.DB) port.CustomerRepository {
	return &CustomerRepo{db: db}
}

func (r *CustomerRepo) FindById(ctx context.Context, orgId string, id string) (domain.Customer, error) {
	var row customerRow
	err := dbFromCtx(ctx, r.db).
		Scopes(OrgScope(orgId)).
		Where("id = ?", id).
		First(&row).Error
	if err != nil {
		return domain.Customer{}, translateErr(err)
	}
	return row.toDomain(), nil
}

// FindByIds batch-loads customers by their IDs within an org. Used by services
// that compose read models to avoid N+1 (e.g. OrderService.ListDetails).
func (r *CustomerRepo) FindByIds(ctx context.Context, orgId string, ids []string) ([]domain.Customer, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var rows []customerRow
	err := dbFromCtx(ctx, r.db).
		Scopes(OrgScope(orgId)).
		Where("id IN ?", ids).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]domain.Customer, len(rows))
	for i, row := range rows {
		out[i] = row.toDomain()
	}
	return out, nil
}

func (r *CustomerRepo) FindByEmail(ctx context.Context, orgId string, email string) (domain.Customer, error) {
	var row customerRow
	err := dbFromCtx(ctx, r.db).
		Scopes(OrgScope(orgId)).
		Where("email = ?", email).
		First(&row).Error
	if err != nil {
		return domain.Customer{}, translateErr(err)
	}
	return row.toDomain(), nil
}

func (r *CustomerRepo) FindByExternalId(ctx context.Context, orgId string, externalId string) (domain.Customer, error) {
	var row customerRow
	err := dbFromCtx(ctx, r.db).
		Scopes(OrgScope(orgId)).
		Where("external_id = ?", externalId).
		First(&row).Error
	if err != nil {
		return domain.Customer{}, translateErr(err)
	}
	return row.toDomain(), nil
}

func (r *CustomerRepo) Create(ctx context.Context, entity domain.Customer) (domain.Customer, error) {
	row := customerRowFromDomain(entity)
	if err := r.writeRow(ctx, &row, false).Error; err != nil {
		return domain.Customer{}, asConflictOnUnique(err, "A customer with this email or external id already exists")
	}
	return r.FindById(ctx, entity.OrgId, entity.Id)
}

func (r *CustomerRepo) Update(ctx context.Context, entity domain.Customer) (domain.Customer, error) {
	row := customerRowFromDomain(entity)
	if err := r.writeRow(ctx, &row, true).Error; err != nil {
		return domain.Customer{}, err
	}
	return r.FindById(ctx, entity.OrgId, entity.Id)
}

// writeRow issues the insert/update for a customer row, omitting
// default_payment_method_id when it is empty. The column is a nullable FK to
// payment_methods; the domain models "no default" as the empty string, so
// writing "" would violate the FK. Omitting it lets the column stay NULL.
func (r *CustomerRepo) writeRow(ctx context.Context, row *customerRow, update bool) *gorm.DB {
	db := dbFromCtx(ctx, r.db)
	if row.DefaultPaymentMethodId == "" {
		db = db.Omit("default_payment_method_id")
	}
	if update {
		return db.Save(row)
	}
	return db.Create(row)
}

func (r *CustomerRepo) List(ctx context.Context, orgId string, pagination domain.Pagination) ([]domain.Customer, int, error) {
	var rows []customerRow
	var count int64
	if err := dbFromCtx(ctx, r.db).Model(&customerRow{}).
		Scopes(OrgScope(orgId)).
		Count(&count).Error; err != nil {
		return nil, 0, err
	}
	if err := dbFromCtx(ctx, r.db).
		Scopes(OrgScope(orgId), Paginate(pagination)).
		Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	out := make([]domain.Customer, len(rows))
	for i, row := range rows {
		out[i] = row.toDomain()
	}
	return out, int(count), nil
}

func (r *CustomerRepo) FindPaymentMethodById(ctx context.Context, orgId string, id string) (domain.PaymentMethod, error) {
	var row paymentMethodRow
	err := dbFromCtx(ctx, r.db).
		Scopes(OrgScope(orgId)).
		Where("id = ?", id).
		First(&row).Error
	if err != nil {
		return domain.PaymentMethod{}, translateErr(err)
	}
	return row.toDomain(), nil
}

func (r *CustomerRepo) AddToCohort(ctx context.Context, orgId string, customerId string, cohortId string, cohortValue string) (domain.Customer, error) {
	cc := customerCohortRow{
		OrgId:       orgId,
		CustomerId:  customerId,
		CohortId:    cohortId,
		CohortValue: cohortValue,
	}
	if err := dbFromCtx(ctx, r.db).Create(&cc).Error; err != nil {
		return domain.Customer{}, err
	}
	return r.FindById(ctx, orgId, customerId)
}

func (r *CustomerRepo) FindCohortById(ctx context.Context, orgId string, id string) (domain.Cohort, error) {
	var row cohortRow
	err := dbFromCtx(ctx, r.db).
		Scopes(OrgScope(orgId)).
		Where("id = ?", id).
		First(&row).Error
	if err != nil {
		return domain.Cohort{}, translateErr(err)
	}
	return row.toDomain(), nil
}

func (r *CustomerRepo) CreateCohort(ctx context.Context, input domain.Cohort) (domain.Cohort, error) {
	row := cohortRowFromDomain(input)
	if err := dbFromCtx(ctx, r.db).Create(&row).Error; err != nil {
		return domain.Cohort{}, err
	}
	return r.FindCohortById(ctx, input.OrgId, input.Id)
}

func (r *CustomerRepo) UpdateCohort(ctx context.Context, input domain.Cohort) (domain.Cohort, error) {
	row := cohortRowFromDomain(input)
	if err := dbFromCtx(ctx, r.db).Save(&row).Error; err != nil {
		return domain.Cohort{}, err
	}
	return r.FindCohortById(ctx, input.OrgId, input.Id)
}

func (r *CustomerRepo) DeleteCohort(ctx context.Context, input domain.Cohort) (domain.Cohort, error) {
	if err := dbFromCtx(ctx, r.db).
		Scopes(OrgScope(input.OrgId)).
		Where("id = ?", input.Id).
		Delete(&cohortRow{}).Error; err != nil {
		return domain.Cohort{}, err
	}
	return input, nil
}
