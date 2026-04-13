package postgres

import (
	"context"

	"gorm.io/gorm"
	"payloop/internal/core/domain"
	"payloop/internal/core/port"
)

type SessionRepo struct {
	db *gorm.DB
}

func NewSessionRepo(db *gorm.DB) port.SessionRepository {
	return &SessionRepo{db: db}
}

func (r *SessionRepo) FindById(ctx context.Context, orgId string, id string) (domain.Session, error) {
	var session domain.Session
	err := r.db.WithContext(ctx).
		Scopes(OrgScope(orgId)).
		Where("id = ?", id).
		First(&session).Error
	return session, err
}

func (r *SessionRepo) Create(ctx context.Context, input domain.Session) (domain.Session, error) {
	err := r.db.WithContext(ctx).Create(&input).Error
	if err != nil {
		return domain.Session{}, err
	}
	return r.FindById(ctx, input.OrgId, input.Id)
}
