package postgrespgx

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
)

type OrderRepo struct {
	pool *pgxpool.Pool
}

func NewOrderRepo(pool *pgxpool.Pool) port.OrderRepository {
	return &OrderRepo{pool: pool}
}

func (r *OrderRepo) FindById(ctx context.Context, orgId string, id string) (domain.Order, error) {
	q := dbFromCtx(ctx, r.pool)
	var row orderRow
	if err := row.scanInto(q.QueryRow(ctx,
		`SELECT `+orderColumns+` FROM orders WHERE org_id = $1 AND id = $2`, orgId, id)); err != nil {
		return domain.Order{}, translateErr(err)
	}
	return row.toDomain(), nil
}

func (r *OrderRepo) FindByIdForUpdate(ctx context.Context, orgId string, id string) (domain.Order, error) {
	q := dbFromCtx(ctx, r.pool)
	var row orderRow
	if err := row.scanInto(q.QueryRow(ctx,
		`SELECT `+orderColumns+` FROM orders WHERE org_id = $1 AND id = $2 FOR UPDATE`, orgId, id)); err != nil {
		return domain.Order{}, translateErr(err)
	}
	return row.toDomain(), nil
}

func (r *OrderRepo) Create(ctx context.Context, entity domain.Order) (domain.Order, error) {
	row := orderRowFromDomain(entity)
	q := dbFromCtx(ctx, r.pool)
	_, err := q.Exec(ctx,
		`INSERT INTO orders (`+orderColumns+`) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
		row.OrgId, row.Id, row.CustomerId, row.Reference, row.Status, row.SessionId,
		row.CartId, row.Currency, row.Total, row.Metadata, row.CreatedAt, row.UpdatedAt, row.PaymentSession)
	if err != nil {
		return domain.Order{}, err
	}
	return r.FindById(ctx, entity.OrgId, entity.Id)
}

func (r *OrderRepo) Update(ctx context.Context, entity domain.Order) (domain.Order, error) {
	row := orderRowFromDomain(entity)
	q := dbFromCtx(ctx, r.pool)
	// payment_session is owned by SetPaymentSession, never the general Update, so
	// a routine order update (e.g. CompleteOrder) cannot clobber a stored session.
	_, err := q.Exec(ctx,
		`UPDATE orders SET customer_id=$3, reference=$4, status=$5, session_id=$6, cart_id=$7,
		        currency=$8, total=$9, metadata=$10, updated_at=$11
		 WHERE org_id=$1 AND id=$2`,
		row.OrgId, row.Id, row.CustomerId, row.Reference, row.Status, row.SessionId,
		row.CartId, row.Currency, row.Total, row.Metadata, row.UpdatedAt)
	if err != nil {
		return domain.Order{}, err
	}
	return r.FindById(ctx, entity.OrgId, entity.Id)
}

// SetPaymentSession persists the PSP payment-session payload onto an existing
// order with a targeted update (payment_session + updated_at only). session is
// always non-nil here; it is wrapped through jsonCol to match the order_row
// jsonb encoding.
func (r *OrderRepo) SetPaymentSession(ctx context.Context, orgId, id string, session any) error {
	q := dbFromCtx(ctx, r.pool)
	_, err := q.Exec(ctx,
		`UPDATE orders SET payment_session=$3, updated_at=$4 WHERE org_id=$1 AND id=$2`,
		orgId, id, newJSON(session), time.Now().UTC())
	return err
}

func (r *OrderRepo) Find(ctx context.Context, orgId string, p domain.Pagination) ([]domain.Order, int, error) {
	q := dbFromCtx(ctx, r.pool)
	var count int64
	if err := q.QueryRow(ctx, `SELECT count(*) FROM orders WHERE org_id = $1`, orgId).Scan(&count); err != nil {
		return nil, 0, err
	}
	rows, err := q.Query(ctx, `SELECT `+orderColumns+` FROM orders WHERE org_id = $1`+paginationClause(p), orgId)
	if err != nil {
		return nil, 0, err
	}
	out, err := r.collectOrders(rows)
	if err != nil {
		return nil, 0, err
	}
	return out, int(count), nil
}

func (r *OrderRepo) FindOrderItemById(ctx context.Context, orgId string, id string) (domain.OrderItem, error) {
	q := dbFromCtx(ctx, r.pool)
	var row orderItemRow
	if err := row.scanInto(q.QueryRow(ctx,
		`SELECT `+orderItemColumns+` FROM order_items WHERE org_id = $1 AND id = $2`, orgId, id)); err != nil {
		return domain.OrderItem{}, translateErr(err)
	}
	return row.toDomain(), nil
}

func (r *OrderRepo) CreateOrderItem(ctx context.Context, entity domain.OrderItem) (domain.OrderItem, error) {
	row := orderItemRowFromDomain(entity)
	q := dbFromCtx(ctx, r.pool)
	_, err := q.Exec(ctx,
		`INSERT INTO order_items (`+orderItemColumns+`)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)`,
		row.OrgId, row.Id, row.OrderId, row.ProductId, row.VariantId, row.PriceId,
		row.SubscriptionId, row.Description, row.Quantity, row.TaxTotal, row.DiscountTotal,
		row.Subtotal, row.Total, row.Metadata, row.CreatedAt, row.UpdatedAt)
	if err != nil {
		return domain.OrderItem{}, err
	}
	return r.FindOrderItemById(ctx, entity.OrgId, entity.Id)
}

func (r *OrderRepo) UpdateOrderItem(ctx context.Context, orderItem domain.OrderItem) (domain.OrderItem, error) {
	row := orderItemRowFromDomain(orderItem)
	q := dbFromCtx(ctx, r.pool)
	_, err := q.Exec(ctx,
		`UPDATE order_items SET order_id=$3, product_id=$4, variant_id=$5, price_id=$6,
		        subscription_id=$7, description=$8, quantity=$9, tax_total=$10, discount_total=$11,
		        sub_total=$12, total=$13, metadata=$14, updated_at=$15
		 WHERE org_id=$1 AND id=$2`,
		row.OrgId, row.Id, row.OrderId, row.ProductId, row.VariantId, row.PriceId,
		row.SubscriptionId, row.Description, row.Quantity, row.TaxTotal, row.DiscountTotal,
		row.Subtotal, row.Total, row.Metadata, row.UpdatedAt)
	if err != nil {
		return domain.OrderItem{}, err
	}
	return r.FindOrderItemById(ctx, orderItem.OrgId, orderItem.Id)
}

func (r *OrderRepo) FindOrderItemsByOrderId(ctx context.Context, orgId string, orderId string) ([]domain.OrderItem, error) {
	q := dbFromCtx(ctx, r.pool)
	rows, err := q.Query(ctx,
		`SELECT `+orderItemColumns+` FROM order_items WHERE org_id = $1 AND order_id = $2`, orgId, orderId)
	if err != nil {
		return nil, err
	}
	return r.collectOrderItems(rows)
}

// FindOrderItemsBySubscriptionId returns the order lines a subscription bills
// (the recurring lines stamped with this subscription's id).
func (r *OrderRepo) FindOrderItemsBySubscriptionId(ctx context.Context, orgId string, subscriptionId string) ([]domain.OrderItem, error) {
	q := dbFromCtx(ctx, r.pool)
	rows, err := q.Query(ctx,
		`SELECT `+orderItemColumns+` FROM order_items WHERE org_id = $1 AND subscription_id = $2`, orgId, subscriptionId)
	if err != nil {
		return nil, err
	}
	return r.collectOrderItems(rows)
}

// collectOrders drains rows into domain orders, closing rows.
func (r *OrderRepo) collectOrders(rows pgx.Rows) ([]domain.Order, error) {
	defer rows.Close()
	var out []domain.Order
	for rows.Next() {
		var row orderRow
		if err := row.scanInto(rows); err != nil {
			return nil, err
		}
		out = append(out, row.toDomain())
	}
	return out, rows.Err()
}

// collectOrderItems drains rows into domain order items, closing rows.
func (r *OrderRepo) collectOrderItems(rows pgx.Rows) ([]domain.OrderItem, error) {
	defer rows.Close()
	var out []domain.OrderItem
	for rows.Next() {
		var row orderItemRow
		if err := row.scanInto(rows); err != nil {
			return nil, err
		}
		out = append(out, row.toDomain())
	}
	return out, rows.Err()
}
