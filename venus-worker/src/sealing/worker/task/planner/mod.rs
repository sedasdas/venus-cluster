use anyhow::Result;

use super::{Event, Failure, State, Task};

mod sealer;
pub use sealer::SealerPlanner;

type ExecResult = Result<Event, Failure>;

macro_rules! plan {
    ($e:expr, $st:expr, $($prev:pat => {$($evt:pat => $next:expr,)+},)*) => {
        match $st {
            $(
                $prev => {
                    match $e {
                        $(
                            $evt => $next,
                        )+
                        _ => return Err(anyhow::anyhow!("unexpected event {:?} for state {:?}", $e, $st)),
                    }
                }
            )*

            other => return Err(anyhow::anyhow!("unexpected state {:?}", other)),
        }
    };
}

pub(self) use plan;

pub trait Planner {
    fn plan(&self, evt: &Event, st: &State) -> Result<State>;
    fn exec<'c, 't>(task: &'t mut Task<'c>) -> Result<Option<Event>, Failure>;
}
