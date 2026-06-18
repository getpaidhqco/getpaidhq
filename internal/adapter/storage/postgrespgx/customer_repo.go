package postgrespgx

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
)

type CustomerRepo struct {
	pool *pgxpool.Pool
}

func NewCustomerRepo(pool *pgxpool.Pool) port.CustomerRepository {
	return &CustomerRepo{pool: pool}
}

func (r *CustomerRepo) FindById(ctx context.Context, orgId, id string) (domain.Customer, error) {
	q := dbFromCtx(ctx, r.pool)
	var row customerRow
	if err := row.scanInto(q.QueryRow(ctx,
		`SELECT `+customerColumns+` FROM customers WHERE org_id = $1 AND id = $2`, orgId, id)); err != nil {
		return domain.Customer{}, translateErr(err)
	}
	return row.toDomain(), nil
}

func (r *CustomerRepo) FindByIds(ctx context.Context, orgId string, ids []string) ([]domain.Customer, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	q := dbFromCtx(ctx, r.pool)
	rows, err := q.Query(ctx,
		`SELECT `+customerColumns+` FROM customers WHERE org_id = $1 AND id = ANY($2)`, orgId, ids)
	if err != nil {
		return nil, err
	}
	return r.collect(rows)
}

func (r *CustomerRepo) FindByEmail(ctx context.Context, orgId, email string) (domain.Customer, error) {
	q := dbFromCtx(ctx, r.pool)
	var row customerRow
	if err := row.scanInto(q.QueryRow(ctx,
		`SELECT `+customerColumns+` FROM customers WHERE org_id = $1 AND email = $2`, orgId, email)); err != nil {
		return domain.Customer{}, translateErr(err)
	}
	return row.toDomain(), nil
}

func (r *CustomerRepo) FindByExternalId(ctx context.Context, orgId, externalId string) (domain.Customer, error) {
	q := dbFromCtx(ctx, r.pool)
	var row customerRow
	if err := row.scanInto(q.QueryRow(ctx,
		`SELECT `+customerColumns+` FROM customers WHERE org_id = $1 AND external_id = $2`, orgId, externalId)); err != nil {
		return domain.Customer{}, translateErr(err)
	}
	return row.toDomain(), nil
}

func (r *CustomerRepo) Create(ctx context.Context, entity domain.Customer) (domain.Customer, error) {
	row := customerRowFromDomain(entity)
	q := dbFromCtx(ctx, r.pool)
	_, err := q.Exec(ctx,
		`INSERT INTO customers (`+customerColumns+`) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
		row.OrgId, row.Id, row.ExternalId, row.FirstName, row.LastName, row.Email, row.Phone,
		row.DefaultPaymentMethodId, row.BillingAddress, row.Metadata, row.CreatedAt, row.UpdatedAt)
	if err != nil {
		return domain.Customer{}, asConflictOnUnique(err, "A customer with this email or external id already exists")
	}
	return r.FindById(ctx, entity.OrgId, entity.Id)
}

func (r *CustomerRepo) Update(ctx context.Context, entity domain.Customer) (domain.Customer, error) {
	row := customerRowFromDomain(entity)
	q := dbFromCtx(ctx, r.pool)
	_, err := q.Exec(ctx,
		`UPDATE customers SET external_id=$3, first_name=$4, last_name=$5, email=$6, phone=$7,
		        default_payment_method_id=$8, billing_address=$9, metadata=$10, updated_at=$11
		 WHERE org_id=$1 AND id=$2`,
		row.OrgId, row.Id, row.ExternalId, row.FirstName, row.LastName, row.Email, row.Phone,
		row.DefaultPaymentMethodId, row.BillingAddress, row.Metadata, row.UpdatedAt)
	if err != nil {
		return domain.Customer{}, asConflictOnUnique(err, "A customer with this email or external id already exists")
	}
	return r.FindById(ctx, entity.OrgId, entity.Id)
}

func (r *CustomerRepo) List(ctx context.Context, orgId string, p domain.Pagination) ([]domain.Customer, int, error) {
	q := dbFromCtx(ctx, r.pool)
	var count int64
	if err := q.QueryRow(ctx, `SELECT count(*) FROM customers WHERE org_id = $1`, orgId).Scan(&count); err != nil {
		return nil, 0, err
	}
	rows, err := q.Query(ctx, `SELECT `+customerColumns+` FROM customers WHERE org_id = $1`+paginationClause(p), orgId)
	if err != nil {
		return nil, 0, err
	}
	out, err := r.collect(rows)
	if err != nil {
		return nil, 0, err
	}
	return out, int(count), nil
}

func (r *CustomerRepo) FindPaymentMethodById(ctx context.Context, orgId, id string) (domain.PaymentMethod, error) {
	q := dbFromCtx(ctx, r.pool)
	var row paymentMethodRow
	if err := row.scanInto(q.QueryRow(ctx,
		`SELECT `+paymentMethodColumns+` FROM payment_methods WHERE org_id = $1 AND id = $2`, orgId, id)); err != nil {
		return domain.PaymentMethod{}, translateErr(err)
	}
	return row.toDomain(), nil
}

func (r *CustomerRepo) AddToCohort(ctx context.Context, orgId, customerId, cohortId, cohortValue string) (domain.Customer, error) {
	q := dbFromCtx(ctx, r.pool)
	_, err := q.Exec(ctx,
		`INSERT INTO customer_cohorts (org_id, customer_id, cohort_id, cohort_value, joined_at, created_at, updated_at)
		 VALUES ($1,$2,$3,$4, now(), now(), now())`,
		orgId, customerId, cohortId, cohortValue)
	if err != nil {
		return domain.Customer{}, err
	}
	return r.FindById(ctx, orgId, customerId)
}

func (r *CustomerRepo) FindCohortById(ctx context.Context, orgId, id string) (domain.Cohort, error) {
	q := dbFromCtx(ctx, r.pool)
	var row cohortRow
	if err := row.scanInto(q.QueryRow(ctx,
		`SELECT `+cohortColumns+` FROM cohorts WHERE org_id = $1 AND id = $2`, orgId, id)); err != nil {
		return domain.Cohort{}, translateErr(err)
	}
	return row.toDomain(), nil
}

func (r *CustomerRepo) CreateCohort(ctx context.Context, input domain.Cohort) (domain.Cohort, error) {
	row := cohortRowFromDomain(input)
	q := dbFromCtx(ctx, r.pool)
	_, err := q.Exec(ctx,
		`INSERT INTO cohorts (`+cohortColumns+`) VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		row.OrgId, row.Id, row.Name, row.Type, row.Metadata, row.CreatedAt, row.UpdatedAt)
	if err != nil {
		return domain.Cohort{}, err
	}
	return r.FindCohortById(ctx, input.OrgId, input.Id)
}

func (r *CustomerRepo) UpdateCohort(ctx context.Context, input domain.Cohort) (domain.Cohort, error) {
	row := cohortRowFromDomain(input)
	q := dbFromCtx(ctx, r.pool)
	_, err := q.Exec(ctx,
		`UPDATE cohorts SET name=$3, type=$4, metadata=$5, updated_at=$6 WHERE org_id=$1 AND id=$2`,
		row.OrgId, row.Id, row.Name, row.Type, row.Metadata, row.UpdatedAt)
	if err != nil {
		return domain.Cohort{}, err
	}
	return r.FindCohortById(ctx, input.OrgId, input.Id)
}

func (r *CustomerRepo) DeleteCohort(ctx context.Context, input domain.Cohort) (domain.Cohort, error) {
	q := dbFromCtx(ctx, r.pool)
	if _, err := q.Exec(ctx, `DELETE FROM cohorts WHERE org_id = $1 AND id = $2`, input.OrgId, input.Id); err != nil {
		return domain.Cohort{}, err
	}
	return input, nil
}

// collect drains rows into domain customers, closing rows.
func (r *CustomerRepo) collect(rows pgx.Rows) ([]domain.Customer, error) {
	defer rows.Close()
	var out []domain.Customer
	for rows.Next() {
		var row customerRow
		if err := row.scanInto(rows); err != nil {
			return nil, err
		}
		out = append(out, row.toDomain())
	}
	return out, rows.Err()
}
