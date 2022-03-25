//! external implementations of processors

use std::sync::atomic::{AtomicUsize, Ordering};

use anyhow::{anyhow, Context, Result};
use crossbeam_channel::{bounded, Sender};

use super::*;

pub mod config;
pub mod sub;

pub struct ExtProcessor<I>
where
    I: Input,
{
    called: AtomicUsize,
    txes: Vec<Sender<(I, Sender<Result<I::Out>>)>>,
}

impl<I> ExtProcessor<I>
where
    I: Input,
{
    pub fn build(cfg: &Vec<config::Ext>) -> Result<(Self, Vec<sub::SubProcess<I>>)> {
        let (txes, subproc) = sub::start_sub_processes(cfg)
            .with_context(|| format!("start sub process for stage {}", I::STAGE.name()))?;

        let proc = Self {
            called: AtomicUsize::new(0),
            txes,
        };

        Ok((proc, subproc))
    }
}

impl<I> Processor<I> for ExtProcessor<I>
where
    I: Input,
{
    fn process(&self, input: I) -> Result<I::Out> {
        let size = self.txes.len();
        if size == 0 {
            return Err(anyhow!("no available sub processor"));
        }

        let called = self.called.fetch_add(1, Ordering::SeqCst) - 1;
        let input_tx = &self.txes[called % size];

        let (res_tx, res_rx) = bounded(1);
        input_tx.send((input, res_tx)).map_err(|e| {
            anyhow!(
                "failed to send input through chan for stage {}: {:?}",
                I::STAGE.name(),
                e
            )
        })?;

        let res = res_rx
            .recv()
            .with_context(|| format!("recv process result for stage {}", I::STAGE.name()))?;

        res
    }
}
