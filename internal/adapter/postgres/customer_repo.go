package postgres

import (
	"context"

	"gorm.io/gorm"
	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
)

type CustomerRepo struct {
	db *gorm.DB
}

func NewCustomerRepo(db *gorm.DB) port.CustomerRepository {
	return &CustomerRepo{db: db}
}

func (r *CustomerRepo) FindById(ctx context.Context, orgId string, id string) (domain.Customer, error) {
	var customer domain.Customer
	err := dbFromCtx(ctx, r.db).
		Scopes(OrgScope(orgId)).
		Where("id = ?", id).
		First(&customer).Error
	return customer, translateErr(err)
}

func (r *CustomerRepo) FindByEmail(ctx context.Context, orgId string, email string) (domain.Customer, error) {
	var customer domain.Customer
	err := dbFromCtx(ctx, r.db).
		Scopes(OrgScope(orgId)).
		Where("email = ?", email).
		First(&customer).Error
	return customer, translateErr(err)
}

func (r *CustomerRepo) Create(ctx context.Context, entity domain.Customer) (domain.Customer, error) {
	err := dbFromCtx(ctx, r.db).Create(&entity).Error
	if err != nil {
		return domain.Customer{}, err
	}
	return r.FindById(ctx, entity.OrgId, entity.Id)
}

func (r *CustomerRepo) Update(ctx context.Context, entity domain.Customer) (domain.Customer, error) {
	err := dbFromCtx(ctx, r.db).Save(&entity).Error
	if err != nil {
		return domain.Customer{}, err
	}
	return r.FindById(ctx, entity.OrgId, entity.Id)
}

func (r *CustomerRepo) List(ctx context.Context, orgId string, pagination domain.Pagination) ([]domain.Customer, int, error) {
	var customers []domain.Customer
	var count int64
	err := dbFromCtx(ctx, r.db).Model(&domain.Customer{}).
		Scopes(OrgScope(orgId)).
		Count(&count).Error
	if err != nil {
		return nil, 0, err
	}
	err = dbFromCtx(ctx, r.db).
		Scopes(OrgScope(orgId), Paginate(pagination)).
		Find(&customers).Error
	return customers, int(count), err
}

func (r *CustomerRepo) FindPaymentMethodById(ctx context.Context, orgId string, id string) (domain.PaymentMethod, error) {
	var pm domain.PaymentMethod
	err := dbFromCtx(ctx, r.db).
		Scopes(OrgScope(orgId)).
		Where("id = ?", id).
		First(&pm).Error
	return pm, translateErr(err)
}

func (r *CustomerRepo) AddToCohort(ctx context.Context, orgId string, customerId string, cohortId string, cohortValue string) (domain.Customer, error) {
	cc := domain.CustomerCohort{
		OrgId:       orgId,
		CustomerId:  customerId,
		CohortId:    cohortId,
		CohortValue: cohortValue,
	}
	err := dbFromCtx(ctx, r.db).Create(&cc).Error
	if err != nil {
		return domain.Customer{}, err
	}
	return r.FindById(ctx, orgId, customerId)
}

func (r *CustomerRepo) FindCohortById(ctx context.Context, orgId string, id string) (domain.Cohort, error) {
	var cohort domain.Cohort
	err := dbFromCtx(ctx, r.db).
		Scopes(OrgScope(orgId)).
		Where("id = ?", id).
		First(&cohort).Error
	return cohort, translateErr(err)
}

func (r *CustomerRepo) CreateCohort(ctx context.Context, input domain.Cohort) (domain.Cohort, error) {
	err := dbFromCtx(ctx, r.db).Create(&input).Error
	if err != nil {
		return domain.Cohort{}, err
	}
	return r.FindCohortById(ctx, input.OrgId, input.Id)
}

func (r *CustomerRepo) UpdateCohort(ctx context.Context, input domain.Cohort) (domain.Cohort, error) {
	err := dbFromCtx(ctx, r.db).Save(&input).Error
	if err != nil {
		return domain.Cohort{}, err
	}
	return r.FindCohortById(ctx, input.OrgId, input.Id)
}

func (r *CustomerRepo) DeleteCohort(ctx context.Context, input domain.Cohort) (domain.Cohort, error) {
	err := dbFromCtx(ctx, r.db).
		Scopes(OrgScope(input.OrgId)).
		Where("id = ?", input.Id).
		Delete(&domain.Cohort{}).Error
	if err != nil {
		return domain.Cohort{}, err
	}
	return input, nil
}
