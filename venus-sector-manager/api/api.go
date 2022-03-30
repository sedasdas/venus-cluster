package api

import (
	"context"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/specs-storage/storage"
	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/venus/venus-shared/actors/builtin"
	"github.com/filecoin-project/venus/venus-shared/types"

	"github.com/ipfs-force-community/venus-cluster/venus-sector-manager/pkg/objstore"
)

const MajorVersion = 0

var Empty Meta

type Meta *struct{}

type SealerAPI interface {
	AllocateSector(context.Context, AllocateSectorSpec) (*AllocatedSector, error)

	AcquireDeals(context.Context, abi.SectorID, AcquireDealsSpec) (Deals, error)

	AssignTicket(context.Context, abi.SectorID) (Ticket, error)

	SubmitPreCommit(context.Context, AllocatedSector, PreCommitOnChainInfo, bool) (SubmitPreCommitResp, error)

	PollPreCommitState(context.Context, abi.SectorID) (PollPreCommitStateResp, error)

	SubmitPersisted(context.Context, abi.SectorID, string) (bool, error)

	WaitSeed(context.Context, abi.SectorID) (WaitSeedResp, error)

	SubmitProof(context.Context, abi.SectorID, ProofOnChainInfo, bool) (SubmitProofResp, error)

	PollProofState(context.Context, abi.SectorID) (PollProofStateResp, error)

	ListSectors(context.Context, SectorWorkerState) ([]*SectorState, error)

	RestoreSector(ctx context.Context, sid abi.SectorID, forced bool) (Meta, error)

	ReportState(context.Context, abi.SectorID, ReportStateReq) (Meta, error)

	ReportFinalized(context.Context, abi.SectorID) (Meta, error)

	ReportAborted(context.Context, abi.SectorID, string) (Meta, error)

	CheckProvable(context.Context, abi.RegisteredPoStProof, []storage.SectorRef, bool) (map[abi.SectorNumber]string, error)

	SimulateWdPoSt(context.Context, address.Address, []builtin.ExtendedSectorInfo, abi.PoStRandomness) error

	// Snap
	AllocateSanpUpSector(ctx context.Context, spec AllocateSnapUpSpec) (*AllocatedSnapUpSector, error)

	SubmitSnapUpProof(ctx context.Context, sid abi.SectorID, pieces []cid.Cid, proof []byte, instance string) (SubmitSnapUpProofResp, error)
}

type RandomnessAPI interface {
	GetTicket(context.Context, types.TipSetKey, abi.ChainEpoch, abi.ActorID) (Ticket, error)
	GetSeed(context.Context, types.TipSetKey, abi.ChainEpoch, abi.ActorID) (Seed, error)
	GetWindowPoStChanlleengeRand(context.Context, types.TipSetKey, abi.ChainEpoch, abi.ActorID) (WindowPoStRandomness, error)
	GetWindowPoStCommitRand(context.Context, types.TipSetKey, abi.ChainEpoch) (WindowPoStRandomness, error)
}

type MinerInfoAPI interface {
	Get(context.Context, abi.ActorID) (*MinerInfo, error)
}

type SectorManager interface {
	Allocate(ctx context.Context, spec AllocateSectorSpec) (*AllocatedSector, error)
}

type DealManager interface {
	Acquire(ctx context.Context, sid abi.SectorID, spec AcquireDealsSpec) (Deals, error)
	Release(ctx context.Context, sid abi.SectorID, acquired Deals) error
}

type CommitmentManager interface {
	SubmitPreCommit(context.Context, abi.SectorID, PreCommitInfo, bool) (SubmitPreCommitResp, error)
	PreCommitState(context.Context, abi.SectorID) (PollPreCommitStateResp, error)

	SubmitProof(context.Context, abi.SectorID, ProofInfo, bool) (SubmitProofResp, error)
	ProofState(context.Context, abi.SectorID) (PollProofStateResp, error)
}

type SectorNumberAllocator interface {
	Next(context.Context, abi.ActorID, uint64, func(uint64) bool) (uint64, bool, error)
}

type SectorStateManager interface {
	Init(context.Context, abi.SectorID, abi.RegisteredSealProof) error
	InitWith(ctx context.Context, sid abi.SectorID, proofType abi.RegisteredSealProof, fields ...interface{}) error
	Load(context.Context, abi.SectorID) (*SectorState, error)
	Update(context.Context, abi.SectorID, ...interface{}) error
	Finalize(context.Context, abi.SectorID, func(*SectorState) error) error
	Restore(context.Context, abi.SectorID, func(*SectorState) error) error
	All(ctx context.Context, ws SectorWorkerState) ([]*SectorState, error)
}

type SectorIndexer interface {
	Find(context.Context, abi.SectorID) (string, bool, error)
	Update(context.Context, abi.SectorID, string) error
	StoreMgr() objstore.Manager
}

type SectorTracker interface {
	Provable(context.Context, abi.RegisteredPoStProof, []storage.SectorRef, bool) (map[abi.SectorNumber]string, error)
	PubToPrivate(context.Context, abi.ActorID, []builtin.ExtendedSectorInfo) (SortedPrivateSectorInfo, error)
}

type SnapUpSectorManager interface {
	PreFetch(ctx context.Context, mid abi.ActorID, dlindex *uint64) (uint64, error)
	Allocate(ctx context.Context, spec AllocateSectorSpec) (*SnapUpCandidate, error)
	Release(ctx context.Context, candidate *SnapUpCandidate) error
}
