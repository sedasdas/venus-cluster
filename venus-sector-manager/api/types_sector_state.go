package api

import (
	"github.com/filecoin-project/go-state-types/abi"
)

type SectorFinalized bool
type SectorUpgraded bool
type SectorUpgradeLanded *abi.ChainEpoch

type SectorState struct {
	ID         abi.SectorID
	SectorType abi.RegisteredSealProof

	// may be nil
	Deals  Deals
	Ticket *Ticket
	Seed   *Seed
	Pre    *PreCommitInfo
	Proof  *ProofInfo

	MessageInfo MessageInfo

	LatestState *ReportStateReq
	Finalized   SectorFinalized
	AbortReason string

	// for snapup
	Upgraded      SectorUpgraded
	UpgradeLanded SectorUpgradeLanded
}

func (s SectorState) DealIDs() []abi.DealID {
	res := make([]abi.DealID, 0, len(s.Deals))
	for i := range s.Deals {
		if id := s.Deals[i].ID; id != 0 {
			res = append(res, id)
		}
	}
	return res
}

type SectorWorkerState string

const (
	WorkerOnline  SectorWorkerState = "online"
	WorkerOffline SectorWorkerState = "offline"
)
