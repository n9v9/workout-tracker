mod dal;
mod server;

use std::{
    net::SocketAddr,
    path::{Path, PathBuf},
};

use argh::FromArgs;
use sqlx::{
    sqlite::{SqliteConnectOptions, SqlitePoolOptions},
    Pool, Sqlite,
};
use tracing::{info, trace};
use tracing_subscriber::EnvFilter;

/// Server binary for the `workout-tracker` application.
#[derive(Debug, FromArgs)]
struct Args {
    /// path to the database file
    #[argh(option)]
    db: PathBuf,

    /// address and port to listen on (default 127.0.0.1:8080)
    #[argh(option, default = "\"127.0.0.1:8080\".parse().unwrap()")]
    addr: SocketAddr,
}

#[tokio::main]
async fn main() {
    setup_tracing();

    let args: Args = argh::from_env();
    trace!(?args, "Parsed CLI arguments.");

    let pool = setup_database(&args.db).await.unwrap();

    server::run(&args.addr, pool).await;
}

fn setup_tracing() {
    if std::env::var("RUST_LOG").is_err() {
        std::env::set_var("RUST_LOG", "server=trace,tower_http=trace");
    }

    tracing_subscriber::fmt()
        .with_env_filter(EnvFilter::from_default_env())
        .init();
}

async fn setup_database(file: &Path) -> sqlx::Result<Pool<Sqlite>> {
    let pool = SqlitePoolOptions::new()
        .connect_with(
            SqliteConnectOptions::new()
                .filename(file)
                .create_if_missing(true)
                .foreign_keys(true),
        )
        .await?;

    info!("Running database migrations.");
    sqlx::migrate!().run(&pool).await?;

    Ok(pool)
}
