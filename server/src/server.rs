use std::net::SocketAddr;

use axum::{
    extract::{Path, State},
    http::{header::CONTENT_TYPE, Request, StatusCode, Uri},
    middleware::{self, Next},
    response::{IntoResponse, Response},
    routing::{get, post},
    Json, Router, Server, ServiceExt,
};
use include_dir::{include_dir, Dir};
use sqlx::{Pool, Sqlite};
use tokio::signal;
use tower::ServiceBuilder;
use tower_http::{
    request_id::MakeRequestUuid,
    trace::{DefaultMakeSpan, TraceLayer},
    ServiceBuilderExt,
};
use tracing::{error, info};

use crate::dal;

use self::{
    requests::{CreateUpdateExercise, CreateUpdateExerciseSet, GetSetSuggestion},
    responses::{Exercise, ExerciseCount, ExerciseSet, SetSuggestion, StatisticsOverview, Workout},
};

static STATIC_FILES: Dir<'_> = include_dir!("../client/dist");

#[derive(Debug, Clone)]
struct AppState {
    pool: Pool<Sqlite>,
}

pub async fn run(addr: &SocketAddr, pool: Pool<Sqlite>) {
    let state = AppState { pool };

    let check_workout_exists_layer =
        || middleware::from_fn_with_state(state.clone(), check_workout_exists);

    let check_exercise_exists_layer =
        || middleware::from_fn_with_state(state.clone(), check_exercise_exists);

    let check_exercise_set_exists_layer =
        || middleware::from_fn_with_state(state.clone(), check_exercise_set_exists);

    let endpoints = Router::new()
        .route("/workouts", get(get_workouts).post(create_workout))
        .route(
            "/workouts/:id",
            get(get_workout)
                .delete(delete_workout)
                .route_layer(check_workout_exists_layer()),
        )
        .route(
            "/workouts/:id/sets",
            get(get_exercise_sets_by_workout_id).route_layer(check_workout_exists_layer()),
        )
        .route("/workouts/:id/sets/suggest", post(get_set_suggestion))
        .route("/exercises", get(get_exercises).post(create_exercise))
        .route(
            "/exercises/:id",
            get(get_exercise)
                .put(update_exercise)
                .delete(delete_exercise)
                .route_layer(check_exercise_exists_layer()),
        )
        .route(
            "/exercises/:id/sets",
            get(get_exercise_sets_by_exercise_id).route_layer(check_exercise_exists_layer()),
        )
        .route(
            "/exercises/:id/count",
            get(get_exercise_count).route_layer(check_exercise_exists_layer()),
        )
        .route("/sets", get(get_exercise_sets).post(create_exercise_set))
        .route(
            "/sets/:id",
            get(get_exercise_set)
                .put(update_exercise_set)
                .delete(delete_exercise_set)
                .route_layer(check_exercise_set_exists_layer()),
        )
        .route("/statistics", get(get_statistics_overview));

    let router = Router::new()
        .nest("/api", endpoints)
        .nest_service("/", get(get_static_file))
        .with_state(state);

    let svc = ServiceBuilder::new()
        .set_x_request_id(MakeRequestUuid)
        .layer(
            TraceLayer::new_for_http()
                .make_span_with(DefaultMakeSpan::default().include_headers(true)),
        )
        .propagate_x_request_id()
        .service(router);

    info!(%addr, "Listening on {}", addr);

    Server::bind(addr)
        .serve(svc.into_make_service())
        .with_graceful_shutdown(shutdown_signal())
        .await
        .unwrap();
}

async fn shutdown_signal() {
    signal::ctrl_c()
        .await
        .expect("failed to install CTRL+C signal handler");

    info!("Shutting down...");
}

async fn get_static_file(uri: Uri) -> Response {
    let path = match uri.path().trim_start_matches('/') {
        "" => "index.html",
        path => path,
    };

    let Some(file) = STATIC_FILES.get_file(path) else {
        return StatusCode::NOT_FOUND.into_response();
    };

    let guess = mime_guess::from_path(path)
        .first_or_text_plain()
        .to_string();

    ([(CONTENT_TYPE, guess)], file.contents()).into_response()
}

async fn check_workout_exists<T>(
    State(state): State<AppState>,
    Path(id): Path<i64>,
    request: Request<T>,
    next: Next<T>,
) -> Response {
    match dal::get_workout(&state.pool, id).await {
        Err(err) => {
            error!(%err, "Failed to check if workout exists.");
            StatusCode::INTERNAL_SERVER_ERROR.into_response()
        }
        Ok(None) => StatusCode::NOT_FOUND.into_response(),
        _ => next.run(request).await,
    }
}

async fn check_exercise_exists<T>(
    State(state): State<AppState>,
    Path(id): Path<i64>,
    request: Request<T>,
    next: Next<T>,
) -> Response {
    match dal::get_exercise(&state.pool, id).await {
        Err(err) => {
            error!(%err, "Failed to check if exercise exists.");
            StatusCode::INTERNAL_SERVER_ERROR.into_response()
        }
        Ok(None) => StatusCode::NOT_FOUND.into_response(),
        _ => next.run(request).await,
    }
}

async fn check_exercise_set_exists<T>(
    State(state): State<AppState>,
    Path(id): Path<i64>,
    request: Request<T>,
    next: Next<T>,
) -> Response {
    match dal::get_exercise_set(&state.pool, id).await {
        Err(err) => {
            error!(%err, "Failed to check if exercise set exists.");
            StatusCode::INTERNAL_SERVER_ERROR.into_response()
        }
        Ok(None) => StatusCode::NOT_FOUND.into_response(),
        _ => next.run(request).await,
    }
}

async fn get_exercise(
    State(state): State<AppState>,
    Path(id): Path<i64>,
) -> Result<Json<Exercise>, AppError> {
    dal::get_exercise(&state.pool, id)
        .await?
        .map(|exercise| Json(Exercise::from(exercise)))
        .ok_or_else(|| AppError::StatusCode(StatusCode::NOT_FOUND))
}

async fn get_exercises(State(state): State<AppState>) -> Result<Json<Vec<Exercise>>, AppError> {
    let exercises = dal::get_exercises(&state.pool)
        .await?
        .into_iter()
        .map(Exercise::from)
        .collect();
    Ok(Json(exercises))
}

async fn create_exercise(
    State(state): State<AppState>,
    Json(exercise): Json<CreateUpdateExercise>,
) -> Result<Json<Exercise>, AppError> {
    let exercise = dal::create_exercise(&state.pool, &exercise.name).await?;
    Ok(Json(Exercise::from(exercise)))
}

async fn update_exercise(
    State(state): State<AppState>,
    Path(id): Path<i64>,
    Json(exercise): Json<CreateUpdateExercise>,
) -> Result<Json<Exercise>, AppError> {
    let exercise = dal::update_exercise(&state.pool, id, &exercise.name).await?;
    Ok(Json(Exercise::from(exercise)))
}

async fn delete_exercise(
    State(state): State<AppState>,
    Path(id): Path<i64>,
) -> Result<StatusCode, AppError> {
    dal::delete_exercise(&state.pool, id)
        .await?
        .map(|_| StatusCode::NO_CONTENT)
        .ok_or_else(|| AppError::StatusCode(StatusCode::NOT_FOUND))
}

async fn get_exercise_count(
    State(state): State<AppState>,
    Path(id): Path<i64>,
) -> Result<Json<responses::ExerciseCount>, AppError> {
    let count = dal::get_exercise_count(&state.pool, id).await?;
    Ok(Json(ExerciseCount::from(count)))
}

async fn get_workout(
    State(state): State<AppState>,
    Path(id): Path<i64>,
) -> Result<Json<Workout>, AppError> {
    dal::get_workout(&state.pool, id)
        .await?
        .map(|workout| Json(Workout::from(workout)))
        .ok_or_else(|| AppError::StatusCode(StatusCode::NOT_FOUND))
}

async fn get_workouts(State(state): State<AppState>) -> Result<Json<Vec<Workout>>, AppError> {
    let workouts = dal::get_workouts(&state.pool)
        .await?
        .into_iter()
        .map(Workout::from)
        .collect();
    Ok(Json(workouts))
}

async fn create_workout(State(state): State<AppState>) -> Result<Json<Workout>, AppError> {
    let workout = dal::create_workout(&state.pool).await?;
    Ok(Json(Workout::from(workout)))
}

async fn delete_workout(
    State(state): State<AppState>,
    Path(id): Path<i64>,
) -> Result<StatusCode, AppError> {
    dal::delete_workout(&state.pool, id)
        .await?
        .map(|_| StatusCode::NO_CONTENT)
        .ok_or_else(|| AppError::StatusCode(StatusCode::NOT_FOUND))
}

async fn get_exercise_set(
    State(state): State<AppState>,
    Path(id): Path<i64>,
) -> Result<Json<ExerciseSet>, AppError> {
    dal::get_exercise_set(&state.pool, id)
        .await?
        .map(|exercise| Json(ExerciseSet::from(exercise)))
        .ok_or_else(|| AppError::StatusCode(StatusCode::NOT_FOUND))
}

async fn get_exercise_sets(
    State(state): State<AppState>,
) -> Result<Json<Vec<ExerciseSet>>, AppError> {
    let exercise_sets = dal::get_exercise_sets(&state.pool)
        .await?
        .into_iter()
        .map(ExerciseSet::from)
        .collect();
    Ok(Json(exercise_sets))
}

async fn get_exercise_sets_by_workout_id(
    State(state): State<AppState>,
    Path(id): Path<i64>,
) -> Result<Json<Vec<ExerciseSet>>, AppError> {
    let exercise_sets = dal::get_exercise_sets_by_workout_id(&state.pool, id)
        .await?
        .into_iter()
        .map(ExerciseSet::from)
        .collect();
    Ok(Json(exercise_sets))
}

async fn get_exercise_sets_by_exercise_id(
    State(state): State<AppState>,
    Path(id): Path<i64>,
) -> Result<Json<Vec<ExerciseSet>>, AppError> {
    let exercise_sets = dal::get_exercise_sets_by_exercise_id(&state.pool, id)
        .await?
        .into_iter()
        .map(ExerciseSet::from)
        .collect();
    Ok(Json(exercise_sets))
}

async fn create_exercise_set(
    State(state): State<AppState>,
    Json(exercise_set): Json<CreateUpdateExerciseSet>,
) -> Result<Json<ExerciseSet>, AppError> {
    let exercise_set = dal::create_or_update_exercise_set(
        &state.pool,
        None,
        exercise_set.workout_id,
        exercise_set.exercise_id,
        exercise_set.repetitions,
        exercise_set.weight,
        exercise_set.note,
    )
    .await?;
    Ok(Json(ExerciseSet::from(exercise_set)))
}

async fn update_exercise_set(
    State(state): State<AppState>,
    Path(id): Path<i64>,
    Json(exercise_set): Json<CreateUpdateExerciseSet>,
) -> Result<Json<ExerciseSet>, AppError> {
    let exercise_set = dal::create_or_update_exercise_set(
        &state.pool,
        Some(id),
        exercise_set.workout_id,
        exercise_set.exercise_id,
        exercise_set.repetitions,
        exercise_set.weight,
        exercise_set.note,
    )
    .await?;
    Ok(Json(ExerciseSet::from(exercise_set)))
}

async fn delete_exercise_set(
    State(state): State<AppState>,
    Path(id): Path<i64>,
) -> Result<StatusCode, AppError> {
    dal::delete_exercise_set(&state.pool, id)
        .await?
        .map(|_| StatusCode::NO_CONTENT)
        .ok_or_else(|| AppError::StatusCode(StatusCode::NOT_FOUND))
}

async fn get_set_suggestion(
    State(state): State<AppState>,
    Path(id): Path<i64>,
    Json(request): Json<GetSetSuggestion>,
) -> Result<Json<SetSuggestion>, AppError> {
    let suggestion =
        dal::get_set_suggestion_for_workout(&state.pool, id, request.exercise_id).await?;
    Ok(Json(SetSuggestion::from(suggestion)))
}

async fn get_statistics_overview(
    State(state): State<AppState>,
) -> Result<Json<StatisticsOverview>, AppError> {
    let overview = dal::get_statistics_overview(&state.pool).await?;
    Ok(Json(StatisticsOverview::from(overview)))
}

#[derive(Debug)]
enum AppError {
    Err(anyhow::Error),
    StatusCode(StatusCode),
}

impl From<anyhow::Error> for AppError {
    fn from(err: anyhow::Error) -> Self {
        Self::Err(err)
    }
}

impl IntoResponse for AppError {
    fn into_response(self) -> Response {
        match self {
            Self::Err(err) => {
                let category = if err.downcast_ref::<sqlx::Error>().is_some() {
                    "Database error."
                } else {
                    "Unknown error."
                };
                error!(err = format!("{err:#}"), "{category}");
                StatusCode::INTERNAL_SERVER_ERROR.into_response()
            }
            Self::StatusCode(status) => status.into_response(),
        }
    }
}

mod requests {
    use serde::{Deserialize, Serialize};

    #[derive(Debug, Serialize, Deserialize)]
    pub struct CreateUpdateExercise {
        pub name: String,
    }

    #[derive(Debug, Serialize, Deserialize)]
    pub struct CreateUpdateExerciseSet {
        #[serde(rename = "workoutId")]
        pub workout_id: i64,
        #[serde(rename = "exerciseId")]
        pub exercise_id: i64,
        pub repetitions: i64,
        pub weight: i64,
        pub note: String,
    }

    #[derive(Debug, Serialize, Deserialize)]
    pub struct GetSetSuggestion {
        #[serde(rename = "exerciseId")]
        pub exercise_id: Option<i64>,
    }
}

mod responses {
    use serde::{Deserialize, Serialize};

    use crate::dal::{
        ExerciseCountEntity, ExerciseEntity, ExerciseSetEntity, SetSuggestionEntity,
        StatisticsOverviewEntity, WorkoutEntity,
    };

    #[derive(Debug, Deserialize, Serialize)]
    pub struct Exercise {
        pub id: i64,
        pub name: String,
    }

    impl From<ExerciseEntity> for Exercise {
        fn from(value: ExerciseEntity) -> Self {
            Self {
                id: value.id,
                name: value.name,
            }
        }
    }

    #[derive(Debug, Deserialize, Serialize)]
    pub struct Workout {
        pub id: i64,
        #[serde(rename = "createdUtcSeconds")]
        pub created_utc_s: i64,
    }

    impl From<WorkoutEntity> for Workout {
        fn from(value: WorkoutEntity) -> Self {
            Self {
                id: value.id,
                created_utc_s: value.started.timestamp(),
            }
        }
    }

    #[derive(Debug, Deserialize, Serialize)]
    pub struct ExerciseSet {
        pub id: i64,
        #[serde(rename = "exerciseId")]
        pub exercise_id: i64,
        #[serde(rename = "exerciseName")]
        pub exercise_name: String,
        #[serde(rename = "workoutId")]
        pub workout_id: i64,
        #[serde(rename = "createdUtcSeconds")]
        pub created_utc_s: i64,
        pub repetitions: i64,
        pub weight: i64,
        pub note: Option<String>,
    }

    impl From<ExerciseSetEntity> for ExerciseSet {
        fn from(value: ExerciseSetEntity) -> Self {
            Self {
                id: value.id,
                exercise_id: value.exercise_id,
                exercise_name: value.exercise_name,
                workout_id: value.workout_id,
                created_utc_s: value.created.timestamp(),
                repetitions: value.repetitions,
                weight: value.weight,
                note: value.note,
            }
        }
    }

    #[derive(Debug, Serialize)]
    pub struct SetSuggestion {
        #[serde(rename = "exerciseId")]
        pub exercise_id: i64,
        pub repetitions: i64,
        pub weight: i64,
    }

    impl From<SetSuggestionEntity> for SetSuggestion {
        fn from(value: SetSuggestionEntity) -> Self {
            Self {
                exercise_id: value.exercise_id,
                repetitions: value.repetitions,
                weight: value.weight,
            }
        }
    }

    #[derive(Debug, Serialize)]
    pub struct ExerciseCount {
        pub count: i64,
    }

    impl From<ExerciseCountEntity> for ExerciseCount {
        fn from(value: ExerciseCountEntity) -> Self {
            Self { count: value.count }
        }
    }

    #[derive(Debug, Serialize)]
    pub struct StatisticsOverview {
        #[serde(rename = "totalWorkouts")]
        total_workouts: i64,
        #[serde(rename = "totalDurationSeconds")]
        total_duration_s: i64,
        #[serde(rename = "avgDurationSeconds")]
        avg_duration_s: i64,
        #[serde(rename = "totalSets")]
        total_sets: i64,
        #[serde(rename = "totalReps")]
        total_repetitions: i64,
        #[serde(rename = "avgRepsPerSet")]
        avg_repetitions_per_set: i64,
    }

    impl From<StatisticsOverviewEntity> for StatisticsOverview {
        fn from(value: StatisticsOverviewEntity) -> Self {
            Self {
                total_workouts: value.total_workouts,
                total_duration_s: value.total_duration_s,
                avg_duration_s: value.avg_duration_s,
                total_sets: value.total_sets,
                total_repetitions: value.total_repetitions,
                avg_repetitions_per_set: value.avg_repetitions_per_set,
            }
        }
    }
}
