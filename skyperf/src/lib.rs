//! skyperf: small, focused perf-sensitive helpers for Sky Panel.
//!
//! This crate is intentionally tiny — it exists to do the handful of things
//! that are painful or slow to do well from Go: recursive directory sizing,
//! streaming tar+zstd backups, and polling log-tail following.

pub mod backup;
pub mod dirsize;
pub mod tail;
