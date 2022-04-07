//! provides logging helpers

use anyhow::{Context, Result};
use crossterm::tty::IsTty;
use tracing_subscriber::{
    filter::{self, FilterExt},
    fmt::{layer, time},
    prelude::*,
    registry,
};

pub use tracing::{
    debug, debug_span, error, error_span, field::debug as debug_field,
    field::display as display_field, info, info_span, trace, trace_span, warn, warn_span, Span,
};

/// initiate the global tracing subscriber
pub fn init() -> Result<()> {
    let env_filter = filter::EnvFilter::builder()
        .with_default_directive(filter::LevelFilter::DEBUG.into())
        .from_env()
        .context("invalid env filter")?
        .add_directive("want=warn".parse()?)
        .add_directive("jsonrpc_client_transports=warn".parse()?)
        .add_directive("jsonrpc_core=warn".parse()?)
        .add_directive("hyper=warn".parse()?)
        .add_directive("mio=warn".parse()?)
        .add_directive("reqwest=warn".parse()?);

    let worker_env_filter = filter::EnvFilter::builder()
        .with_default_directive(filter::LevelFilter::OFF.into())
        .with_env_var("VENUS_WORKER_LOG")
        .from_env()
        .context("invalid venus worker log filter")?;

    let fmt_layer = layer()
        .with_writer(std::io::stderr)
        .with_ansi(std::io::stderr().is_tty())
        .with_target(true)
        .with_thread_ids(true)
        .with_timer(time::LocalTime::rfc_3339())
        .with_filter(env_filter.or(worker_env_filter));

    registry().with(fmt_layer).init();

    Ok(())
}
