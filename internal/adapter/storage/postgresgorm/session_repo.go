package postgresgorm

import (
	"context"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"

	"gorm.io/gorm"
)

type SessionRepo struct {
	db *gorm.DB
}

func NewSessionRepo(db *gorm.DB) port.SessionRepository {
	return &SessionRepo{db: db}
}

func (r *SessionRepo) FindById(ctx context.Context, orgId string, id string) (domain.Session, error) {
	var row sessionRow
	err := dbFromCtx(ctx, r.db).
		Scopes(OrgScope(orgId)).
		Where("id = ?", id).
		First(&row).Error
	if err != nil {
		return domain.Session{}, translateErr(err)
	}
	return row.toDomain(), nil
}

func (r *SessionRepo) Create(ctx context.Context, input domain.Session) (domain.Session, error) {
	row := sessionRowFromDomain(input)
	if err := dbFromCtx(ctx, r.db).Create(&row).Error; err != nil {
		return domain.Session{}, err
	}
	return r.FindById(ctx, input.OrgId, input.Id)
}
