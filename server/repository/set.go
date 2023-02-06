package repository

import (
	"context"
	"strings"

	"github.com/jmoiron/sqlx"
)

type SetRepository interface {
	// ByID returns the set with the given ID.
	//
	// # Errors
	//
	// Returns either [database/sql.ErrNoRows] or another, underlying SQL error.
	ByID(ctx context.Context, id int64) (SetEntity, error)

	// ByWorkoutID returns all sets that belong to the workout with the given ID.
	//
	// # Errors
	//
	// Returns an underlying SQL error.
	ByWorkoutID(ctx context.Context, id int64) ([]SetEntity, error)

	// Create creates a set with the given values.
	//
	// # Errors
	//
	// Returns an underlying SQL error.
	Create(ctx context.Context, data CreateSetEntity) error

	// Update updates the set with the given ID.
	//
	// # Errors
	//
	// Returns an underlying SQL error.
	Update(ctx context.Context, data UpdateSetEntity) error

	// Delete tries to delete a set with the given ID.
	//
	// # Errors
	//
	// Returns an underlying SQL error.
	Delete(ctx context.Context, id int64) error
}

type SetEntity struct {
	ID                   int64   `db:"id"`
	ExerciseID           int64   `db:"exercise_id"`
	ExerciseName         string  `db:"exercise_name"`
	DoneSecondsUnixEpoch int     `db:"done_seconds_unix_epoch"`
	Repetitions          int     `db:"repetitions"`
	Weight               int     `db:"weight"`
	Note                 *string `db:"note"`
}

type UpdateSetEntity struct {
	ID          int64
	ExerciseID  int64
	Repetitions int
	Weight      int
	Note        string
}

type CreateSetEntity struct {
	WorkoutID   int64
	ExerciseID  int64
	Repetitions int
	Weight      int
	Note        string
}

type setRepository struct {
	db *sqlx.DB
}

func NewSetRepository(db *sqlx.DB) SetRepository {
	return &setRepository{db}
}

func (sr *setRepository) ByID(ctx context.Context, id int64) (SetEntity, error) {
	const query = `
		SELECT es.id,
			   es.exercise_id,
			   e.name                 AS exercise_name,
			   UNIXEPOCH(es.date_utc) AS done_seconds_unix_epoch,
			   es.repetitions,
			   es.weight,
			   es.note
		  FROM exercise_set AS es
			   JOIN
			   exercise     AS e ON es.exercise_id = e.id
		 WHERE es.id = ?
		 ORDER BY es.date_utc DESC
	`

	var entity SetEntity

	if err := sr.db.GetContext(ctx, &entity, query, id); err != nil {
		return entity, err
	}

	return entity, nil
}

func (sr *setRepository) ByWorkoutID(ctx context.Context, id int64) ([]SetEntity, error) {
	const query = `
		SELECT es.id,
			   es.exercise_id,
			   e.name                 AS exercise_name,
			   UNIXEPOCH(es.date_utc) AS done_seconds_unix_epoch,
			   es.repetitions,
			   es.weight,
			   es.note
		  FROM exercise_set AS es
			   JOIN
			   exercise     AS e ON es.exercise_id = e.id
		 WHERE es.workout_id = ?
		 ORDER BY es.date_utc DESC
	`

	var entities []SetEntity

	if err := sr.db.SelectContext(ctx, &entities, query, id); err != nil {
		return nil, err
	}

	return entities, nil
}

func (sr *setRepository) Create(ctx context.Context, data CreateSetEntity) error {
	const query = `
		INSERT INTO exercise_set (exercise_id,
								  workout_id,
								  date_utc,
								  repetitions,
								  weight,
		                          note)
		VALUES (?,
				?,
				DATETIME('now'),
				?,
				?,
		        ?)
	`

	var trimmedNote *string

	if v := strings.TrimSpace(data.Note); v != "" {
		trimmedNote = &v
	}

	_, err := sr.db.ExecContext(
		ctx, query, data.ExerciseID, data.WorkoutID, data.Repetitions, data.Weight, trimmedNote,
	)

	return err
}

func (sr *setRepository) Update(ctx context.Context, data UpdateSetEntity) error {
	const query = `
		UPDATE
			exercise_set
		   SET exercise_id = ?,
			   repetitions = ?,
			   weight      = ?,
			   note        = ?
		 WHERE id = ?
	`

	var trimmedNote *string

	if v := strings.TrimSpace(data.Note); v != "" {
		trimmedNote = &v
	}

	if _, err := sr.db.ExecContext(ctx, query, data.ExerciseID, data.Repetitions, data.Weight, trimmedNote, data.ID); err != nil {
		return err
	}

	return nil
}

func (sr *setRepository) Delete(ctx context.Context, id int64) error {
	const query = `
		DELETE
		  FROM exercise_set
		 WHERE id = ?
	`

	if _, err := sr.db.ExecContext(ctx, query, id); err != nil {
		return err
	}

	return nil
}
