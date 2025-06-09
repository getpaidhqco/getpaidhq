package postgres

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/repositories"
	"payloop/internal/lib"
)

type DocSequenceRepository struct {
	*PgDatabase
	logger logger.Logger
}

func NewDocSequenceRepository(primaryDb lib.Database, logger logger.Logger) repositories.DocSequenceRepository {
	pgDatabase, ok := primaryDb.(*PgDatabase)
	if !ok {
		panic("database is not of type *db.PgDatabase")
	}
	return DocSequenceRepository{
		PgDatabase: pgDatabase,
		logger:     logger,
	}
}

// WithTrx enables repository with transaction
func (r DocSequenceRepository) WithTrx(trxHandle interface{}) DocSequenceRepository {
	if trxHandle == nil {
		r.logger.Warn("Transaction Database not found in gin context. ")
		return r
	}
	r.PgDatabase.Tx = trxHandle.(pgx.Tx)
	return r
}

func (r DocSequenceRepository) FindById(ctx context.Context, orgId string, id string) (entities.DocSequence, error) {
	tx := r.getTransactionFromContext(ctx)

	var sequence entities.DocSequence

	query := `SELECT org_id, id, type, value, created_at, updated_at
			  FROM "doc_sequences"
			  WHERE org_id = $1 AND id = $2`

	err := tx.QueryRow(ctx, query, orgId, id).Scan(
		&sequence.OrgId,
		&sequence.Id,
		&sequence.Type,
		&sequence.Value,
		&sequence.CreatedAt,
		&sequence.UpdatedAt,
	)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			r.logger.Error("failed to find DocSequence", "err", pgErr.Message, "code", pgErr.Code)
		}
		if errors.Is(err, pgx.ErrNoRows) {
			r.logger.Error("DocSequence not found")
		}
		return entities.DocSequence{}, err
	}

	return sequence, nil
}

func (r DocSequenceRepository) FindByType(ctx context.Context, orgId string, sequenceType string) ([]entities.DocSequence, error) {
	tx := r.getTransactionFromContext(ctx)

	var sequences = make([]entities.DocSequence, 0)

	query := `SELECT org_id, id, type, value, created_at, updated_at
			  FROM "doc_sequences"
			  WHERE org_id = $1 AND type = $2
			  ORDER BY value DESC`

	rows, err := tx.Query(ctx, query, orgId, sequenceType)

	if err != nil {
		r.logger.Error(`failed to find DocSequences by type`, err.Error())
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var sequence entities.DocSequence

		err := rows.Scan(
			&sequence.OrgId,
			&sequence.Id,
			&sequence.Type,
			&sequence.Value,
			&sequence.CreatedAt,
			&sequence.UpdatedAt,
		)

		if err != nil {
			r.logger.Error(`failed to scan DocSequence`, err.Error())
			return nil, err
		}

		sequences = append(sequences, sequence)
	}

	if rows.Err() != nil {
		r.logger.Error(`rows iteration error`, rows.Err().Error())
		return nil, rows.Err()
	}

	return sequences, nil
}

func (r DocSequenceRepository) Create(ctx context.Context, entity entities.DocSequence) (entities.DocSequence, error) {
	tx := r.getTransactionFromContext(ctx)

	query := `INSERT INTO "doc_sequences" (org_id, id, type, value, created_at, updated_at)
			  VALUES (@org_id, @id, @type, @value, NOW(), NOW())`

	_, err := tx.Exec(ctx, query, pgx.NamedArgs{
		"org_id": entity.OrgId,
		"id":     entity.Id,
		"type":   entity.Type,
		"value":  entity.Value,
	})

	if err != nil {
		r.logger.Error(`failed to insert DocSequence`, err.Error())
		return entities.DocSequence{}, err
	}

	return r.FindById(ctx, entity.OrgId, entity.Id)
}

func (r DocSequenceRepository) Update(ctx context.Context, entity entities.DocSequence) (entities.DocSequence, error) {
	tx := r.getTransactionFromContext(ctx)

	query := `UPDATE "doc_sequences"
			  SET type = @type, value = @value, updated_at = NOW()
			  WHERE org_id = @org_id AND id = @id`

	_, err := tx.Exec(ctx, query, pgx.NamedArgs{
		"org_id": entity.OrgId,
		"id":     entity.Id,
		"type":   entity.Type,
		"value":  entity.Value,
	})

	if err != nil {
		r.logger.Error(`failed to update DocSequence`, err.Error())
		return entities.DocSequence{}, err
	}

	return r.FindById(ctx, entity.OrgId, entity.Id)
}

func (r DocSequenceRepository) GetNextValue(ctx context.Context, orgId string, id string, sequenceType string) (int, error) {
	tx := r.getTransactionFromContext(ctx)

	// First try to find the sequence
	var sequence entities.DocSequence
	var nextValue int

	findQuery := `SELECT org_id, id, type, value, created_at, updated_at
				  FROM "doc_sequences"
				  WHERE org_id = $1 AND id = $2 AND type = $3
				  FOR UPDATE`

	err := tx.QueryRow(ctx, findQuery, orgId, id, sequenceType).Scan(
		&sequence.OrgId,
		&sequence.Id,
		&sequence.Type,
		&sequence.Value,
		&sequence.CreatedAt,
		&sequence.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Sequence doesn't exist, create it with initial value 1
			newSequence := entities.DocSequence{
				OrgId: orgId,
				Id:    id,
				Type:  sequenceType,
				Value: 1,
			}
			_, err = r.Create(ctx, newSequence)
			if err != nil {
				r.logger.Error(`failed to create new DocSequence`, err.Error())
				return 0, err
			}
			return 1, nil
		}
		r.logger.Error(`failed to find DocSequence for update`, err.Error())
		return 0, err
	}

	// Increment the value
	nextValue = sequence.Value + 1
	updateQuery := `UPDATE "doc_sequences"
				   SET value = $1, updated_at = NOW()
				   WHERE org_id = $2 AND id = $3 AND type = $4`

	_, err = tx.Exec(ctx, updateQuery, nextValue, orgId, id, sequenceType)
	if err != nil {
		r.logger.Error(`failed to update DocSequence value`, err.Error())
		return 0, err
	}

	return nextValue, nil
}
