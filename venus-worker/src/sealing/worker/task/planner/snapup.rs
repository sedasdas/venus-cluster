use std::io::{self, prelude::*};
use std::os::unix::fs::symlink;

use anyhow::{anyhow, Context, Result};

use super::{
    super::{call_rpc, cloned_required, field_required, Finalized},
    common, plan, Event, ExecResult, Planner, State, Task,
};
use crate::logging::{debug, warn};
use crate::rpc::sealer::{AcquireDealsSpec, AllocateSectorSpec, AllocateSnapUpSpec, SubmitResult};
use crate::sealing::failure::*;
use crate::sealing::processor::{
    snap_generate_partition_proofs, snap_verify_sector_update_proof, tree_d_path_in_dir,
    write_and_preprocess, SnapEncodeInput, SnapProveInput, UnpaddedBytesAmount,
};

pub struct SnapUpPlanner;

impl Planner for SnapUpPlanner {
    fn plan(&self, evt: &Event, st: &State) -> Result<State> {
        let next = plan! {
            evt,
            st,

            State::Empty => {
                Event::AllocatedSnapUpSector(_, _, _) => State::Allocated,
            },

            State::Allocated => {
                Event::AddPiece(_) => State::PieceAdded,
            },

            State::PieceAdded => {
                Event::BuildTreeD => State::TreeDBuilt,
            },

            State::TreeDBuilt => {
                Event::SnapEncode(_) => State::SnapEncoded,
            },

            State::SnapEncoded => {
                Event::SnapProve(_) => State::SnapProved,
            },

            State::SnapProved => {
                Event::Persist(_) => State::Persisted,
            },

            State::Persisted => {
                Event::RePersist => State::SnapProved,
                Event::Finish => State::Finished,
            },
        };

        Ok(next)
    }

    fn exec<'c, 't>(&self, task: &'t mut Task<'c>) -> Result<Option<Event>, Failure> {
        let state = task.sector.state;
        let inner = SnapUp { task };
        match state {
            State::Empty => inner.empty(),

            State::Allocated => inner.add_piece(),

            State::PieceAdded => inner.build_tree_d(),

            State::TreeDBuilt => inner.snap_encode(),

            State::SnapEncoded => inner.snap_prove(),

            State::SnapProved => inner.persist(),

            State::Persisted => inner.submit(),

            State::Finished => return Ok(None),

            State::Aborted => {
                warn!("sector aborted");
                return Ok(None);
            }

            other => return Err(anyhow!("unexpected state {:?} in snapup planner", other).abort()),
        }
        .map(From::from)
    }
}

struct SnapUp<'c, 't> {
    task: &'t mut Task<'c>,
}

impl<'c, 't> SnapUp<'c, 't> {
    fn empty(&self) -> ExecResult {
        let maybe_res = call_rpc! {
            self.task.ctx.global.rpc,
            allocate_snapup_sector,
            AllocateSnapUpSpec {
                sector: AllocateSectorSpec {
                    allowed_miners: self.task.store.allowed_miners.as_ref().cloned(),
                    allowed_proof_types: self.task.store.allowed_proof_types.as_ref().cloned(),
                },
                deals: AcquireDealsSpec {
                    max_deals: None,
                },
            },
        };

        let maybe_allocated = match maybe_res {
            Ok(a) => a,
            Err(e) => {
                warn!("sectors are not allocated yet, so we can retry even though we got the err {:?}", e);
                return Ok(Event::Retry);
            }
        };

        let allocated = match maybe_allocated {
            Some(a) => a,
            None => return Ok(Event::Retry),
        };

        if allocated.deals.is_empty() {
            return Err(anyhow!("deals required, got empty").abort());
        }

        if allocated.private.access_instance.is_empty() {
            return Err(anyhow!("access instance required, got empty").abort());
        }

        Ok(Event::AllocatedSnapUpSector(
            allocated.sector,
            allocated.deals,
            Finalized {
                public: allocated.public,
                private: allocated.private,
            },
        ))
    }

    fn add_piece(&self) -> ExecResult {
        let sector_id = self.task.sector_id()?;
        let proof_type = self.task.sector_proof_type()?;

        let mut staged_file = self.task.staged_file(sector_id).init_file().perm()?;

        let piece_store = self
            .task
            .ctx
            .global
            .piece_store
            .as_ref()
            .context("piece store is required")
            .perm()?;

        field_required!(deals, self.task.sector.deals.as_ref());
        let seal_proof_type = proof_type.into();

        let mut pieces = Vec::new();

        for deal in deals {
            debug!(deal_id = deal.id, cid = %deal.piece.cid.0, payload_size = deal.payload_size, piece_size = deal.piece.size.0, "trying to add piece");

            let unpadded_piece_size = deal.piece.size.unpadded();
            let (piece_info, _) = if deal.id == 0 {
                let mut pledge_piece = io::repeat(0).take(unpadded_piece_size.0);
                write_and_preprocess(
                    seal_proof_type,
                    &mut pledge_piece,
                    &mut staged_file,
                    UnpaddedBytesAmount(unpadded_piece_size.0),
                )
                .with_context(|| format!("write pledge piece, size={}", unpadded_piece_size.0))
                .perm()?
            } else {
                let mut piece_reader = piece_store
                    .get(deal.piece.cid.0, deal.payload_size, unpadded_piece_size)
                    .perm()?;

                write_and_preprocess(
                    seal_proof_type,
                    &mut piece_reader,
                    &mut staged_file,
                    UnpaddedBytesAmount(unpadded_piece_size.0),
                )
                .with_context(|| {
                    format!(
                        "write deal piece, cid={}, size={}",
                        deal.piece.cid.0, unpadded_piece_size.0
                    )
                })
                .perm()?
            };

            pieces.push(piece_info);
        }

        Ok(Event::AddPiece(pieces))
    }

    fn build_tree_d(&self) -> ExecResult {
        common::build_tree_d(self.task, false)?;
        Ok(Event::BuildTreeD)
    }

    fn snap_encode(&self) -> ExecResult {
        let sector_id = self.task.sector_id()?;
        let proof_type = self.task.sector_proof_type()?;
        field_required!(
            access_instance,
            self.task
                .sector
                .finalized
                .as_ref()
                .map(|f| &f.private.access_instance)
        );
        cloned_required!(piece_infos, self.task.sector.phases.pieces);

        let access_store = self
            .task
            .ctx
            .global
            .attached
            .get(access_instance.as_str())
            .with_context(|| format!("get access store instance named {}", access_instance))
            .perm()?;

        // sealed file & persisted cache files should be accessed inside persist store
        // TODO: make link type configurable
        let sealed_file = self.task.sealed_file(sector_id);
        sealed_file.prepare().perm()?;
        access_store
            .link_object(sealed_file.rel(), sealed_file.as_ref(), true)
            .perm()?;

        let cache_dir = self.task.cache_dir(sector_id);
        access_store
            .link_dir(cache_dir.rel(), cache_dir.as_ref(), true)
            .perm()?;

        // init update file
        let update_file = self.task.update_file(sector_id);
        {
            let file = update_file.init_file().perm()?;
            file.set_len(proof_type.sector_size())
                .context("fallocate for update file")
                .perm()?;
        }

        let update_cache_dir = self.task.update_cache_dir(sector_id);
        update_cache_dir.prepare().perm()?;

        // tree d
        let prepared_dir = self.task.prepared_dir(sector_id);
        symlink(
            tree_d_path_in_dir(prepared_dir.as_ref()),
            tree_d_path_in_dir(update_cache_dir.as_ref()),
        )
        .crit()?;

        // staged file shoud be already exists, do nothing
        let staged_file = self.task.staged_file(sector_id);

        let snap_encode_out = self
            .task
            .ctx
            .global
            .processors
            .snap_encode
            .process(SnapEncodeInput {
                registered_proof: proof_type.into(),
                new_replica_path: update_file.into(),
                new_cache_path: update_cache_dir.into(),
                sector_path: sealed_file.into(),
                sector_cache_path: cache_dir.into(),
                staged_data_path: staged_file.into(),
                piece_infos,
            })
            .perm()?;

        Ok(Event::SnapEncode(snap_encode_out))
    }

    fn snap_prove(&self) -> ExecResult {
        let sector_id = self.task.sector_id()?;
        let proof_type = self.task.sector_proof_type()?;
        field_required!(encode_out, self.task.sector.phases.snap_encode_out.as_ref());
        field_required!(
            comm_r_old,
            self.task.sector.finalized.as_ref().map(|f| f.public.comm_r)
        );

        let sealed_file = self.task.sealed_file(sector_id);
        let cached_dir = self.task.cache_dir(sector_id);
        let update_file = self.task.update_file(sector_id);
        let update_cache_dir = self.task.update_cache_dir(sector_id);

        let vannilla_proofs = snap_generate_partition_proofs(
            proof_type.clone().into(),
            comm_r_old,
            encode_out.comm_r_new,
            encode_out.comm_d_new,
            sealed_file.into(),
            cached_dir.into(),
            update_file.into(),
            update_cache_dir.into(),
        )
        .perm()?;

        let proof = self
            .task
            .ctx
            .global
            .processors
            .snap_prove
            .process(SnapProveInput {
                registered_proof: proof_type.clone().into(),
                vannilla_proofs: vannilla_proofs.into_iter().map(|b| b.0).collect(),
                comm_r_old,
                comm_r_new: encode_out.comm_r_new,
                comm_d_new: encode_out.comm_d_new,
            })
            .perm()?;

        let verified = snap_verify_sector_update_proof(
            proof_type.into(),
            &proof,
            comm_r_old,
            encode_out.comm_r_new,
            encode_out.comm_d_new,
        )
        .perm()?;

        if !verified {
            return Err(anyhow!("generated an invalid update proof").perm());
        }

        Ok(Event::SnapProve(proof))
    }

    fn persist(&self) -> ExecResult {
        let sector_id = self.task.sector_id()?;
        let update_cache_dir = self.task.update_cache_dir(sector_id);
        let update_file = self.task.update_file(sector_id);

        let ins_name = common::persist_sector_files(self.task, update_cache_dir, update_file)?;

        Ok(Event::Persist(ins_name))
    }

    fn submit(&self) -> ExecResult {
        let sector_id = self.task.sector_id()?;
        field_required!(proof, self.task.sector.phases.snap_prov_out.as_ref());
        field_required!(deals, self.task.sector.deals.as_ref());
        let piece_cids = deals.iter().map(|d| d.piece.cid.clone()).collect();

        let res = call_rpc! {
            self.task.ctx.global.rpc,
            submit_snapup_proof,
            sector_id.clone(),
            piece_cids,
            proof.into(),
        }?;

        match res.res {
            SubmitResult::Accepted | SubmitResult::DuplicateSubmit => Ok(Event::Finish),

            SubmitResult::MismatchedSubmission => Err(anyhow!(
                "submission for {} is not matched with a previous one: {:?}",
                self.task.sector_path(sector_id),
                res.desc
            )
            .abort()),

            SubmitResult::Rejected => Err(anyhow!("{:?}: {:?}", res.res, res.desc)).abort(),

            SubmitResult::FilesMissed => Ok(Event::RePersist),
        }
    }
}
