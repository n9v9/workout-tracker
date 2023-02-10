package repository

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/jmoiron/sqlx"
)

var ErrExerciseExists = errors.New("exercise exists in at least one set")

type ExerciseRepository interface {
	// FindAll returns all exercises.
	//
	// # Errors
	//
	// Returns an underlying SQL error.
	FindAll(ctx context.Context) ([]ExerciseEntity, error)

	// UsageInSets returns the number of times the exercise with
	// the given id is used in sets.
	//
	// # Errors
	//
	// Returns an underlying SQL error.
	UsageInSets(ctx context.Context, id int64) (int64, error)

	// ExistsID checks whether an exercise with the given id exists.
	//
	// # Errors
	//
	// Returns an underlying SQL error.
	ExistsID(ctx context.Context, id int64) (bool, error)

	// ExistsName returns whether an exercise with the given name exists.
	//
	// # Errors
	//
	// Returns an underlying SQL error.
	ExistsName(ctx context.Context, name string) (bool, error)

	// Create creates an exercise with the given name.
	//
	// # Errors
	//
	// Returns an underlying SQL error.
	Create(ctx context.Context, name string) (ExerciseEntity, error)

	// Update changes the name of an existing exercise.
	//
	// # Errors
	//
	// Returns an underlying SQL error.
	Update(ctx context.Context, id int64, name string) (ExerciseEntity, error)

	// Delete deletes the exercise with the given id.
	// If the exercise is used in any sets, errExerciseExists will be returned.
	//
	// # Errors
	//
	// Returns errExerciseExists if the exercise exists, or an underlying SQL error.
	Delete(ctx context.Context, id int64) error
}

type ExerciseEntity struct {
	ID   int64  `db:"id"`
	Name string `db:"name"`
}

type exerciseRepository struct {
	db *sqlx.DB
}

func NewExerciseRepository(db *sqlx.DB) ExerciseRepository {
	return &exerciseRepository{db}
}

func (er *exerciseRepository) FindAll(ctx context.Context) ([]ExerciseEntity, error) {
	const query = `
               SELECT id,
                          name
                 FROM exercise
                ORDER BY name
       `

	var exercises []ExerciseEntity

	if err := er.db.SelectContext(ctx, &exercises, query); err != nil {
		return nil, err
	}

	return exercises, nil

}

func (er *exerciseRepository) UsageInSets(ctx context.Context, id int64) (int64, error) {
	const checkQuery = `
		SELECT COUNT(*)
		  FROM exercise     e
			   JOIN
			   exercise_set es ON e.id = es.exercise_id
		 WHERE e.id = ?;
	`

	var count int64

	err := er.db.GetContext(ctx, &count, checkQuery, id)
	if err != nil {
		return 0, err
	}

	return count, nil
}

func (er *exerciseRepository) ExistsID(ctx context.Context, id int64) (bool, error) {
	const query = `
		SELECT 1
		  FROM exercise
		 WHERE id = ?
	`

	// Don't care about this value, just care about the existence.
	var tmp string

	err := er.db.QueryRowxContext(ctx, query, id).Scan(&tmp)

	if err == nil {
		return true, nil
	}

	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}

	return false, err
}

func (er *exerciseRepository) ExistsName(ctx context.Context, name string) (bool, error) {
	const query = `
		SELECT 1
		  FROM exercise
		 WHERE LOWER(name) = LOWER(?)
	`

	// Don't care about this value, just care about the existence.
	var tmp string

	err := er.db.QueryRowxContext(ctx, query, strings.TrimSpace(name)).Scan(&tmp)

	if err == nil {
		return true, nil
	}

	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}

	return false, err
}

func (er *exerciseRepository) Create(ctx context.Context, name string) (ExerciseEntity, error) {
	const query = `
		INSERT INTO exercise (name)
		VALUES (?)
	`

	result, err := er.db.ExecContext(ctx, query, strings.TrimSpace(name))
	if err != nil {
		return ExerciseEntity{}, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return ExerciseEntity{}, err
	}

	return ExerciseEntity{ID: id, Name: name}, nil
}

func (er *exerciseRepository) Update(ctx context.Context, id int64, name string) (ExerciseEntity, error) {
	const query = `
		UPDATE exercise
		   SET name = ?
		 WHERE id = ?
	`

	_, err := er.db.ExecContext(ctx, query, strings.TrimSpace(name), id)
	if err != nil {
		return ExerciseEntity{}, err
	}

	return ExerciseEntity{ID: id, Name: name}, nil
}

func (er *exerciseRepository) Delete(ctx context.Context, id int64) error {
	const checkQuery = `
		SELECT COUNT(*)
		  FROM exercise     e
			   JOIN
			   exercise_set es ON e.id = es.exercise_id
		 WHERE e.id = ?;
	`

	var count int64
	err := er.db.GetContext(ctx, &count, checkQuery, id)
	if err != nil {
		return err
	}
	if count > 0 {
		return ErrExerciseExists
	}

	const deleteQuery = `
		DELETE
		  FROM exercise
		 WHERE id = ?
	`
	_, err = er.db.ExecContext(ctx, deleteQuery, id)
	return err
}
