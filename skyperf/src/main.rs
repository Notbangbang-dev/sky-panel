use std::path::PathBuf;
use std::process::ExitCode;

use clap::{Parser, Subcommand};
use serde_json::json;

use skyperf::{backup, dirsize, tail};

#[derive(Parser)]
#[command(
    name = "skyperf",
    about = "Small perf-sensitive helpers for Sky Panel (dirsize, backup, tail)"
)]
struct Cli {
    #[command(subcommand)]
    command: Command,
}

#[derive(Subcommand)]
enum Command {
    /// Recursively compute the total size in bytes under <path>.
    Dirsize { path: PathBuf },

    /// Create or restore zstd-compressed tar backups.
    Backup {
        #[command(subcommand)]
        action: BackupAction,
    },

    /// Print the last 200 lines of a text file, optionally following it.
    Tail {
        path: PathBuf,

        /// Keep running and stream newly appended lines as they arrive.
        #[arg(long)]
        follow: bool,
    },
}

#[derive(Subcommand)]
enum BackupAction {
    /// Create a zstd-compressed tar archive of <src_dir> at <dest>.
    Create { src_dir: PathBuf, dest: PathBuf },

    /// Restore a zstd-compressed tar archive into <dest_dir>.
    Restore { archive: PathBuf, dest_dir: PathBuf },
}

fn main() -> ExitCode {
    let cli = Cli::parse();

    match cli.command {
        Command::Dirsize { path } => run_dirsize(&path),
        Command::Backup { action } => match action {
            BackupAction::Create { src_dir, dest } => run_backup_create(&src_dir, &dest),
            BackupAction::Restore { archive, dest_dir } => run_backup_restore(&archive, &dest_dir),
        },
        Command::Tail { path, follow } => run_tail_cmd(&path, follow),
    }
}

fn run_dirsize(path: &PathBuf) -> ExitCode {
    match dirsize::compute_dir_size(path) {
        Ok(bytes) => {
            print_json(&json!({ "path": path.display().to_string(), "bytes": bytes }));
            ExitCode::SUCCESS
        }
        Err(err) => fail(&err.to_string()),
    }
}

fn run_backup_create(src_dir: &PathBuf, dest: &PathBuf) -> ExitCode {
    match backup::create_backup(src_dir, dest) {
        Ok(bytes) => {
            print_json(&json!({ "created": dest.display().to_string(), "bytes": bytes }));
            ExitCode::SUCCESS
        }
        Err(err) => fail(&err.to_string()),
    }
}

fn run_backup_restore(archive: &PathBuf, dest_dir: &PathBuf) -> ExitCode {
    match backup::restore_backup(archive, dest_dir) {
        Ok(()) => {
            print_json(&json!({ "restored": dest_dir.display().to_string() }));
            ExitCode::SUCCESS
        }
        Err(err) => fail(&err.to_string()),
    }
}

fn run_tail_cmd(path: &PathBuf, follow: bool) -> ExitCode {
    match tail::run_tail(path, follow) {
        Ok(()) => ExitCode::SUCCESS,
        Err(err) => fail(&err.to_string()),
    }
}

fn print_json(value: &serde_json::Value) {
    println!("{}", value);
}

fn fail(message: &str) -> ExitCode {
    println!("{}", json!({ "error": message }));
    ExitCode::FAILURE
}
