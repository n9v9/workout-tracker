package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jmoiron/sqlx"
)

type StatisticsRepository interface {
	// Overview returns basic statistics to provide a simple overview over all workouts.
	//
	// # Errors
	//
	// Returns either [database/sql.ErrNoRows] or another, underlying SQL error.
	Overview(ctx context.Context) (OverviewEntity, error)
}

type OverviewEntity struct {
	TotalWorkouts int64
	TotalDuration time.Duration
	AvgDuration   time.Duration
	TotalReps     int64
	TotalSets     int64
	AvgRepsPerSet int64
}

type statisticsRepository struct {
	db *sqlx.DB
}

func NewStatisticsRepository(db *sqlx.DB) StatisticsRepository {
	return &statisticsRepository{db}
}

func (sr *statisticsRepository) Overview(ctx context.Context) (OverviewEntity, error) {
	const datesQuery = `
		SELECT UNIXEPOCH(w.start_date_utc) AS start_date_utc,
			   UNIXEPOCH(MAX(es.date_utc)) AS end_date_utc
		  FROM exercise_set es
			   JOIN
			   workout      w ON es.workout_id = w.id
		 GROUP BY w.id
	`

	type datesRow struct {
		StartUTC int64 `db:"start_date_utc"`
		EndUTC   int64 `db:"end_date_utc"`
	}

	var workouts []datesRow

	if err := sr.db.SelectContext(ctx, &workouts, datesQuery); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return OverviewEntity{}, nil
		}
		return OverviewEntity{}, err
	}

	result := OverviewEntity{
		TotalWorkouts: int64(len(workouts)),
	}

	for _, v := range workouts {
		result.TotalDuration += time.Unix(v.EndUTC, 0).Sub(time.Unix(v.StartUTC, 0))
	}

	result.AvgDuration = time.Duration(int64(result.TotalDuration) / result.TotalWorkouts)

	const setsRepsQuery = `
		SELECT COUNT(id)                    AS total_sets,
			   SUM(repetitions)             AS total_reps,
			   SUM(repetitions) / COUNT(id) AS avg_reps_per_set
		  FROM exercise_set;
	`

	type setsRepsRow struct {
		TotalSets     int64 `db:"total_sets"`
		TotalReps     int64 `db:"total_reps"`
		AvgRepsPerSet int64 `db:"avg_reps_per_set"`
	}

	var setsRepsResult setsRepsRow

	if err := sr.db.GetContext(ctx, &setsRepsResult, setsRepsQuery); err != nil {
		return OverviewEntity{}, err
	}

	result.TotalSets = setsRepsResult.TotalSets
	result.TotalReps = setsRepsResult.TotalReps
	result.AvgRepsPerSet = setsRepsResult.AvgRepsPerSet

	return result, nil
}
