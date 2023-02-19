use anyhow::{Context, Result};
use chrono::{DateTime, Duration, Utc};
use sqlx::{FromRow, Pool, Sqlite};

#[derive(Debug, FromRow)]
pub struct ExerciseEntity {
    pub id: i64,
    pub name: String,
}

#[derive(Debug, FromRow)]
pub struct WorkoutEntity {
    pub id: i64,
    #[sqlx(rename = "started_utc_s")]
    pub started: chrono::DateTime<chrono::Utc>,
}

#[derive(Debug, FromRow)]
pub struct SetRecommendationEntity {
    pub exercise_id: i64,
    pub repetitions: i64,
    pub weight: i64,
}

#[derive(Debug, FromRow)]
pub struct ExerciseSetEntity {
    pub id: i64,
    pub exercise_id: i64,
    pub exercise_name: String,
    pub workout_id: i64,
    #[sqlx(rename = "created_utc_s")]
    pub created: DateTime<Utc>,
    pub repetitions: i64,
    pub weight: i64,
    pub note: Option<String>,
}

#[derive(Debug, FromRow)]
pub struct ExerciseCountEntity {
    pub count: i64,
}

#[derive(Debug, Default, FromRow)]
pub struct StatisticsOverviewEntity {
    pub total_workouts: i64,
    pub total_duration_s: i64,
    pub avg_duration_s: i64,
    pub total_sets: i64,
    pub total_repetitions: i64,
    pub avg_repetitions_per_set: i64,
}

pub async fn get_exercise_count(pool: &Pool<Sqlite>, id: i64) -> Result<ExerciseCountEntity> {
    sqlx::query_as("SELECT COUNT(*) AS count FROM exercise_set WHERE exercise_id = ?")
        .bind(id)
        .fetch_one(pool)
        .await
        .with_context(|| format!("Failed to get exercise count for exercise with id {id}"))
}

pub async fn get_exercise(pool: &Pool<Sqlite>, id: i64) -> Result<Option<ExerciseEntity>> {
    sqlx::query_as("SELECT id, name FROM exercise WHERE id = ?")
        .bind(id)
        .fetch_optional(pool)
        .await
        .with_context(|| format!("Failed to get exercise with id {id}"))
}

pub async fn get_exercises(pool: &Pool<Sqlite>) -> Result<Vec<ExerciseEntity>> {
    sqlx::query_as("SELECT id, name FROM exercise ORDER BY name")
        .fetch_all(pool)
        .await
        .context("Failed to get exercises")
}

pub async fn create_exercise(pool: &Pool<Sqlite>, name: &str) -> Result<ExerciseEntity> {
    sqlx::query_as("INSERT INTO exercise (name) VALUES (?) RETURNING id, name")
        .bind(name)
        .fetch_one(pool)
        .await
        .with_context(|| format!(r#"Failed to create exercise with name "{name}""#))
}

pub async fn delete_exercise(pool: &Pool<Sqlite>, id: i64) -> Result<Option<()>> {
    sqlx::query("DELETE FROM exercise WHERE id = ?")
        .bind(id)
        .execute(pool)
        .await
        .map(|res| (res.rows_affected() > 0).then_some(()))
        .with_context(|| format!("Failed to delete exercise with id {id}"))
}

pub async fn update_exercise(pool: &Pool<Sqlite>, id: i64, name: &str) -> Result<ExerciseEntity> {
    sqlx::query_as("UPDATE exercise SET name = ? WHERE id = ? RETURNING id, name")
        .bind(name)
        .bind(id)
        .fetch_one(pool)
        .await
        .with_context(|| format!(r#"Failed to update name of exercise with id {id} to "{name}""#))
}

pub async fn get_workout(pool: &Pool<Sqlite>, id: i64) -> Result<Option<WorkoutEntity>> {
    sqlx::query_as("SELECT id, started_utc_s FROM workout WHERE id = ?")
        .bind(id)
        .fetch_optional(pool)
        .await
        .with_context(|| format!("Failed to get workout with id {id}"))
}

pub async fn get_workouts(pool: &Pool<Sqlite>) -> Result<Vec<WorkoutEntity>> {
    sqlx::query_as("SELECT id, started_utc_s FROM workout")
        .fetch_all(pool)
        .await
        .context("Failed to get workouts")
}

pub async fn create_workout(pool: &Pool<Sqlite>) -> Result<WorkoutEntity> {
    sqlx::query_as(
        "
        INSERT INTO workout (started_utc_s) VALUES (UNIXEPOCH(datetime()))
        RETURNING id, started_utc_s
        ",
    )
    .fetch_one(pool)
    .await
    .context("Failed to create workout")
}

pub async fn delete_workout(pool: &Pool<Sqlite>, id: i64) -> Result<Option<()>> {
    sqlx::query("DELETE FROM workout WHERE id = ?")
        .bind(id)
        .execute(pool)
        .await
        .with_context(|| format!("Failed to delete workout with id {id}"))
        .map(|res| (res.rows_affected() > 0).then_some(()))
}

enum ExerciseSetConstraint {
    ExerciseSetId,
    WorkoutId,
}

fn create_get_exercise_query(constraint: Option<ExerciseSetConstraint>) -> String {
    const GET_ALL_EXERCISES_QUERY: &str = "
    SELECT
        es.id, es.exercise_id, e.name AS exercise_name,
        es.workout_id, es.created_utc_s, es.repetitions, es.weight, es.note
    FROM exercise_set es
    JOIN exercise e ON es.exercise_id = e.id
";

    match constraint {
        Some(ExerciseSetConstraint::ExerciseSetId) => {
            format!("{GET_ALL_EXERCISES_QUERY} WHERE es.id = ?")
        }
        Some(ExerciseSetConstraint::WorkoutId) => {
            format!("{GET_ALL_EXERCISES_QUERY} WHERE es.workout_id = ?")
        }
        None => GET_ALL_EXERCISES_QUERY.to_string(),
    }
}

pub async fn get_exercise_set(pool: &Pool<Sqlite>, id: i64) -> Result<Option<ExerciseSetEntity>> {
    sqlx::query_as(&create_get_exercise_query(Some(
        ExerciseSetConstraint::ExerciseSetId,
    )))
    .bind(id)
    .fetch_optional(pool)
    .await
    .with_context(|| format!("Failed to get exercise set with id {id}"))
}

pub async fn get_exercise_sets(pool: &Pool<Sqlite>) -> Result<Vec<ExerciseSetEntity>> {
    sqlx::query_as(&create_get_exercise_query(None))
        .fetch_all(pool)
        .await
        .context("Failed to get all exercise sets")
}

pub async fn get_exercise_sets_by_workout_id(
    pool: &Pool<Sqlite>,
    id: i64,
) -> Result<Vec<ExerciseSetEntity>> {
    sqlx::query_as(&create_get_exercise_query(Some(
        ExerciseSetConstraint::WorkoutId,
    )))
    .bind(id)
    .fetch_all(pool)
    .await
    .with_context(|| format!("Failed to get exercise sets for workout with id {id}"))
}

pub async fn create_or_update_exercise_set(
    pool: &Pool<Sqlite>,
    exercise_set_id: Option<i64>,
    workout_id: i64,
    exercise_id: i64,
    repetitions: i64,
    weight: i64,
    note: String,
) -> Result<ExerciseSetEntity> {
    let query = match exercise_set_id {
        Some(_) => {
            "
            UPDATE exercise_set
            SET workout_id = ?, exercise_id = ?, repetitions = ?, weight = ?, note = ?
            WHERE id = ?
            RETURNING id, exercise_id, workout_id, created_utc_s, repetitions, weight, note,
                '' AS exercise_name
            "
        }
        None => {
            "
            INSERT INTO exercise_set (workout_id, exercise_id, repetitions, weight, note, created_utc_s)
            VALUES (?, ?, ?, ?, ?, UNIXEPOCH(datetime()))
            RETURNING id, exercise_id, workout_id, created_utc_s, repetitions, weight, note,
                '' AS exercise_name
            "
        }
    };

    // Empty notes are stored as NULL in the database.
    let note = match note.trim() {
        "" => None,
        note => Some(note),
    };

    let mut query = sqlx::query_as::<_, ExerciseSetEntity>(query)
        .bind(workout_id)
        .bind(exercise_id)
        .bind(repetitions)
        .bind(weight)
        .bind(note);

    if let Some(id) = exercise_set_id {
        query = query.bind(id);
    }

    let mut exercise_set = query
        .fetch_one(pool)
        .await
        .with_context(|| {
            format!("Failed to create exercise set with workout id {workout_id} and exercise id {exercise_id}")
        })?;

    exercise_set.exercise_name = get_exercise(pool, exercise_id)
        .await?
        .expect("Exercise must exist as it is used as a foreign key in the previous query")
        .name;

    Ok(exercise_set)
}

pub async fn delete_exercise_set(pool: &Pool<Sqlite>, id: i64) -> Result<Option<()>> {
    sqlx::query("DELETE FROM exercise_set WHERE id = ?")
        .bind(id)
        .execute(pool)
        .await
        .map(|res| (res.rows_affected() > 0).then_some(()))
        .with_context(|| format!("Failed to delete exercise set with id {id}"))
}

pub async fn get_set_recommendation_for_workout(
    pool: &Pool<Sqlite>,
    id: i64,
) -> Result<SetRecommendationEntity> {
    // Just recommend the last set again.
    let recommendation = sqlx::query_as::<_, SetRecommendationEntity>(
        "
        SELECT exercise_id, repetitions, weight
        FROM exercise_set
        WHERE workout_id = ?
        ORDER BY created_utc_s DESC
        LIMIT 1
        ",
    )
    .bind(id)
    .fetch_optional(pool)
    .await?;

    if let Some(set) = recommendation {
        return Ok(set);
    }

    // Suggest the first set of the last workout that has sets.
    let recommendation = sqlx::query_as::<_, SetRecommendationEntity>(
        "
        SELECT exercise_id, repetitions, weight
        FROM exercise_set
        WHERE workout_id = (
            SELECT MAX(w.id)
            FROM workout w
            JOIN exercise_set es ON w.id = es.workout_id
        )
        ORDER BY created_utc_s
        LIMIT 1
        ",
    )
    .bind(id)
    .fetch_optional(pool)
    .await?;

    if let Some(set) = recommendation {
        return Ok(set);
    }

    // Just return some sane defaults.
    Ok(SetRecommendationEntity {
        exercise_id: 0,
        repetitions: 0,
        weight: 0,
    })
}

pub async fn get_statistics_overview(pool: &Pool<Sqlite>) -> Result<StatisticsOverviewEntity> {
    #[derive(Debug, FromRow)]
    struct DatesRow {
        start_utc_s: i64,
        end_utc_s: i64,
    }

    let workouts = sqlx::query_as::<_, DatesRow>(
        "
        SELECT w.started_utc_s AS start_utc_s, MAX(es.created_utc_s) AS end_utc_s
        FROM exercise_set es
        JOIN workout w on es.workout_id = w.id
        GROUP BY w.id
        ",
    )
    .fetch_all(pool)
    .await?;

    if workouts.is_empty() {
        return Ok(Default::default());
    }

    let mut overview = StatisticsOverviewEntity {
        total_workouts: workouts.len() as i64,
        total_duration_s: workouts.iter().map(|w| w.end_utc_s - w.start_utc_s).sum(),
        ..Default::default()
    };

    overview.avg_duration_s = overview.total_duration_s / overview.total_workouts;

    #[derive(Debug, FromRow)]
    struct SetsRepsRow {
        total_sets: i64,
        total_repetitions: i64,
        avg_repetitions_per_set: i64,
    }

    let sets_reps = sqlx::query_as::<_, SetsRepsRow>(
        "
        SELECT
            COUNT(id) AS total_sets,
            SUM(repetitions) AS total_repetitions,
            CAST(AVG(repetitions) AS INT) AS avg_repetitions_per_set
        FROM exercise_set
        ",
    )
    .fetch_one(pool)
    .await?;

    overview.total_sets = sets_reps.total_sets;
    overview.total_repetitions = sets_reps.total_repetitions;
    overview.avg_repetitions_per_set = sets_reps.avg_repetitions_per_set;

    Ok(overview)
}
