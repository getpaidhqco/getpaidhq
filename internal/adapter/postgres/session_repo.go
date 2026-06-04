package postgres

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
	var session domain.Session
	err := dbFromCtx(ctx, r.db).
		Scopes(OrgScope(orgId)).
		Where("id = ?", id).
		First(&session).Error
	return session, translateErr(err)
}

func (r *SessionRepo) Create(ctx context.Context, input domain.Session) (domain.Session, error) {
	err := dbFromCtx(ctx, r.db).Create(&input).Error
	if err != nil {
		return domain.Session{}, err
	}
	return r.FindById(ctx, input.OrgId, input.Id)
}
