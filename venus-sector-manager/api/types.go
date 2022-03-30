package api

import (
	"github.com/filecoin-project/go-address"
	commcid "github.com/filecoin-project/go-fil-commcid"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/venus/venus-shared/actors/builtin/miner"
)

type AllocateSectorSpec struct {
	AllowedMiners     []abi.ActorID
	AllowedProofTypes []abi.RegisteredSealProof
}

type LocatedSector struct {
	DeadlineIndex uint64
	AllocatedSector
}

type AllocatedSector struct {
	ID        abi.SectorID
	ProofType abi.RegisteredSealProof
}

type PieceInfo struct {
	Size abi.PaddedPieceSize
	Cid  cid.Cid
}

type DealInfo struct {
	ID          abi.DealID
	PayloadSize uint64
	Piece       PieceInfo
}

type Deals []DealInfo

type AcquireDealsSpec struct {
	MaxDeals *uint
}

type Ticket struct {
	Ticket abi.Randomness
	Epoch  abi.ChainEpoch
}

type SubmitResult uint64

const (
	SubmitUnknown SubmitResult = iota
	SubmitAccepted
	SubmitDuplicateSubmit
	// worker should enter perm err
	SubmitMismatchedSubmission
	// worker should enter perm err
	SubmitRejected
	// worker should retry persisting files
	SubmitFilesMissed
)

type OnChainState uint64

const (
	OnChainStateUnknown OnChainState = iota
	OnChainStatePending
	OnChainStatePacked
	OnChainStateLanded
	OnChainStateNotFound
	// worker whould try re-submit the info
	OnChainStateFailed
	// worker should enter perm err
	OnChainStatePermFailed
)

type PreCommitOnChainInfo struct {
	CommR  [32]byte
	CommD  [32]byte
	Ticket Ticket
	Deals  []abi.DealID
}

func (pi PreCommitOnChainInfo) IntoPreCommitInfo() (PreCommitInfo, error) {
	commR, err := commcid.ReplicaCommitmentV1ToCID(pi.CommR[:])
	if err != nil {
		return PreCommitInfo{}, err
	}

	commD, err := commcid.DataCommitmentV1ToCID(pi.CommD[:])
	if err != nil {
		return PreCommitInfo{}, err
	}

	return PreCommitInfo{
		CommR:  commR,
		CommD:  commD,
		Ticket: pi.Ticket,
		Deals:  pi.Deals,
	}, nil
}

type ProofOnChainInfo struct {
	Proof []byte
}

type SubmitPreCommitResp struct {
	Res  SubmitResult
	Desc *string
}

type PollPreCommitStateResp struct {
	State OnChainState
	Desc  *string
}

type WaitSeedResp struct {
	ShouldWait bool
	Delay      int
	Seed       *Seed
}

type Seed struct {
	Seed  abi.Randomness
	Epoch abi.ChainEpoch
}

type SubmitProofResp struct {
	Res  SubmitResult
	Desc *string
}

type PollProofStateResp struct {
	State OnChainState
	Desc  *string
}

type MinerInfo struct {
	ID   abi.ActorID
	Addr address.Address
	// Addr                address.Address
	// Owner               address.Address
	// Worker              address.Address
	// NewWorker           address.Address
	// ControlAddresses    []address.Address
	// WorkerChangeEpoch   abi.ChainEpoch
	SectorSize          abi.SectorSize
	WindowPoStProofType abi.RegisteredPoStProof
	SealProofType       abi.RegisteredSealProof
}

type TipSetToken []byte

type PreCommitInfo struct {
	CommR  cid.Cid
	CommD  cid.Cid
	Ticket Ticket
	Deals  []abi.DealID
}

type ProofInfo = ProofOnChainInfo

type AggregateInput struct {
	Spt   abi.RegisteredSealProof
	Info  AggregateSealVerifyInfo
	Proof []byte
}

type PreCommitEntry struct {
	Deposit abi.TokenAmount
	Pci     *miner.SectorPreCommitInfo
}

type MessageInfo struct {
	PreCommitCid *cid.Cid
	CommitCid    *cid.Cid
	NeedSend     bool
}

type ReportStateReq struct {
	Worker      WorkerIdentifier
	StateChange SectorStateChange
	Failure     *SectorFailure
}

type WorkerIdentifier struct {
	Instance string
	Location string
}

type SectorStateChange struct {
	Prev  string
	Next  string
	Event string
}

type SectorFailure struct {
	Level string
	Desc  string
}

type WindowPoStRandomness struct {
	Epoch abi.ChainEpoch
	Rand  abi.Randomness
}

type ActorIdent struct {
	ID   abi.ActorID
	Addr address.Address
}

type AllocateSnapUpSpec struct {
	Sector AllocateSectorSpec
	Deals  AcquireDealsSpec
}

type SectorPublicInfo struct {
	CommR [32]byte
}

type SectorPrivateInfo struct {
	AccessInstance string
}

type AllocatedSnapUpSector struct {
	Sector  AllocatedSector
	Deals   Deals
	Public  SectorPublicInfo
	Private SectorPrivateInfo
}

type SubmitSnapUpProofResp struct {
	Res  SubmitResult
	Desc *string
}
