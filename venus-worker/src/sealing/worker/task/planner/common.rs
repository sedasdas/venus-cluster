//! this module provides some common handlers

use std::fs::{remove_file, File, OpenOptions};
use std::io::{self, prelude::*};
use std::os::unix::fs::symlink;

use anyhow::Context;

use super::super::{Entry, Stage, Task};
use crate::logging::{debug, warn_span};
use crate::rpc::sealer::Deals;
use crate::sealing::failure::*;
use crate::sealing::processor::{
    tree_d_path_in_dir, write_and_preprocess, PieceInfo, RegisteredSealProof, TreeDInput,
    UnpaddedBytesAmount,
};

pub fn add_pieces<'c, 't>(
    task: &'t Task<'c>,
    seal_proof_type: RegisteredSealProof,
    mut staged_file: &mut File,
    deals: &Deals,
) -> Result<Vec<PieceInfo>, Failure> {
    let piece_store = task
        .ctx
        .global
        .piece_store
        .as_ref()
        .context("piece store is required")
        .perm()?;

    let mut pieces = Vec::new();

    for deal in deals {
        debug!(deal_id = deal.id, cid = %deal.piece.cid.0, payload_size = deal.payload_size, piece_size = deal.piece.size.0, "trying to add piece");

        let unpadded_piece_size = deal.piece.size.unpadded();
        let is_pledged = deal.id == 0;
        let (piece_info, _) = if is_pledged {
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

    Ok(pieces)
}

// build tree_d inside `prepare_dir` if necessary
pub fn build_tree_d<'c, 't>(task: &'t Task<'c>, allow_static: bool) -> Result<(), Failure> {
    let sector_id = task.sector_id()?;
    let proof_type = task.sector_proof_type()?;

    let token = task.ctx.global.limit.acquire(Stage::TreeD).crit()?;

    let prepared_dir = task.prepared_dir(sector_id);
    prepared_dir.prepare().perm()?;

    let tree_d_path = tree_d_path_in_dir(prepared_dir.as_ref());
    if tree_d_path.exists() {
        remove_file(&tree_d_path)
            .with_context(|| format!("cleanup preprared tree d file {:?}", tree_d_path))
            .crit()?;
    }

    // pledge sector
    if allow_static && task.sector.deals.as_ref().map(|d| d.len()).unwrap_or(0) == 0 {
        if let Some(static_tree_path) = task.ctx.global.static_tree_d.get(&proof_type.sector_size())
        {
            symlink(static_tree_path, tree_d_path_in_dir(prepared_dir.as_ref())).crit()?;
            return Ok(());
        }
    }

    let staged_file = task.staged_file(sector_id);

    task.ctx
        .global
        .processors
        .tree_d
        .process(TreeDInput {
            registered_proof: proof_type.clone().into(),
            staged_file: staged_file.into(),
            cache_dir: prepared_dir.into(),
        })
        .perm()?;

    drop(token);
    Ok(())
}

// acquire a persist store for sector files, copy the files and return the instance name of the
// acquired store
pub fn persist_sector_files<'c, 't>(
    task: &'t Task<'c>,
    cache_dir: Entry,
    sealed_file: Entry,
) -> Result<String, Failure> {
    let proof_type = task.sector_proof_type()?;
    let sector_size = proof_type.sector_size();

    let persist_store = task
        .ctx
        .global
        .attached
        .acquire_persist(sector_size, None)
        .context("no available persist store")
        .perm()?;

    let ins_name = persist_store.instance();
    debug!(name = ins_name.as_str(), "persist store acquired");

    let mut wanted = vec![sealed_file];

    // here we treat fs err as temp
    for entry_res in cache_dir.read_dir().temp()? {
        let entry = entry_res.temp()?;
        if let Some(fname_str) = entry.rel().file_name().and_then(|name| name.to_str()) {
            let should =
                fname_str == "p_aux" || fname_str == "t_aux" || fname_str.contains("tree-r-last");

            if !should {
                continue;
            }

            wanted.push(entry);
        }
    }

    let mut opt = OpenOptions::new();
    opt.read(true);

    for one in wanted {
        let target_path = one.rel();

        let copy_span = warn_span!(
            "persist",
            src = ?&one,
            dst = ?&target_path,
        );

        let copy_enter = copy_span.enter();

        let source = opt.open(&one).crit()?;
        let size = persist_store.put(target_path, Box::new(source)).crit()?;

        debug!(size, "persist done");

        drop(copy_enter);
    }

    Ok(ins_name)
}
