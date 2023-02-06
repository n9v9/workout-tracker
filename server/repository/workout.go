package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"
)

type WorkoutRepository interface {
	// Create creates a new workout.
	//
	// # Errors
	//
	// Returns an underlying SQL error.
	Create(ctx context.Context) (int64, error)

	// Delete tries to delete the workout with the given ID.
	//
	// # Errors
	//
	// Returns either [database/sql.ErrNoRows] or another, underlying SQL error.
	Delete(ctx context.Context, id int64) error

	// Exists checks whether a workout with the given ID exist.
	//
	// # Errors
	//
	// Returns an underlying SQL error.
	Exists(ctx context.Context, id int64) (bool, error)

	// All returns all workouts.
	//
	// # Errors
	//
	// Returns an underlying SQL error.
	All(ctx context.Context) ([]WorkoutEntity, error)

	// RecommendNewSet returns recommended values for a new set.
	//
	// # Errors
	//
	// Returns an underlying SQL error.
	RecommendNewSet(ctx context.Context, id int64) (SetRecommendationEntity, error)
}

type WorkoutEntity struct {
	ID                    uint64 `db:"id"`
	StartSecondsUnixEpoch uint64 `db:"start_seconds_unix_epoch"`
}

type SetRecommendationEntity struct {
	ExerciseID  int64 `db:"exercise_id"`
	Repetitions int   `db:"repetitions"`
	Weight      int   `db:"weight"`
}

type workoutRepository struct {
	db *sqlx.DB
}

func NewWorkoutRepository(db *sqlx.DB) WorkoutRepository {
	return &workoutRepository{db}
}

func (wr *workoutRepository) Create(ctx context.Context) (int64, error) {
	const query = `
		INSERT INTO workout (start_date_utc)
		VALUES (DATETIME('now'))
	`

	result, err := wr.db.ExecContext(ctx, query)
	if err != nil {
		return 0, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	return id, nil
}

func (wr *workoutRepository) Delete(ctx context.Context, id int64) error {
	const query = `
		DELETE
		  FROM workout
		 WHERE id = ?
	`

	result, err := wr.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func (wr *workoutRepository) Exists(ctx context.Context, id int64) (bool, error) {
	const query = `
		SELECT COUNT(id)
		  FROM workout
		 WHERE id = ?
	`

	var count int

	if err := wr.db.GetContext(ctx, &count, query, id); err != nil {
		return false, err
	}

	return count == 1, nil
}

func (wr *workoutRepository) All(ctx context.Context) ([]WorkoutEntity, error) {
	const query = `
		SELECT id,
			   UNIXEPOCH(start_date_utc) AS start_seconds_unix_epoch
		  FROM workout
		 ORDER BY start_date_utc DESC
	`

	var entities []WorkoutEntity

	if err := wr.db.SelectContext(ctx, &entities, query); err != nil {
		return nil, err
	}

	return entities, nil
}

func (wr *workoutRepository) RecommendNewSet(ctx context.Context, id int64) (SetRecommendationEntity, error) {
	// Very simple recommendation, just recommend the last set.
	const lastSetQuery = `
		SELECT exercise_id,
			   repetitions,
			   weight
		  FROM exercise_set
		 WHERE workout_id = ?
		 ORDER BY date_utc DESC
		 LIMIT 1
	`

	var recommendation SetRecommendationEntity

	err := wr.db.GetContext(ctx, &recommendation, lastSetQuery, id)
	if err == nil {
		return recommendation, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return recommendation, err
	}

	// Suggest the first set of the last workout that has sets.
	const firstSetQuery = `
		SELECT exercise_id,
			   repetitions,
			   weight
		  FROM exercise_set
		 WHERE workout_id = (SELECT MAX(w.id)
							   FROM workout           w
									JOIN exercise_set es ON w.id = es.workout_id)
		 ORDER BY date_utc
		 LIMIT 1;
	`

	err = wr.db.GetContext(ctx, &recommendation, firstSetQuery)
	if err == nil {
		return recommendation, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return recommendation, err
	}

	// There are no workouts with sets, so we just set some defaults.
	recommendation.ExerciseID = -1
	recommendation.Repetitions = 0
	recommendation.Weight = 0

	return recommendation, nil
}
