package sectors

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/filecoin-project/go-state-types/abi"

	"github.com/ipfs-force-community/venus-cluster/venus-sector-manager/api"
	"github.com/ipfs-force-community/venus-cluster/venus-sector-manager/modules"
	"github.com/ipfs-force-community/venus-cluster/venus-sector-manager/pkg/logging"
)

var log = logging.New("sectors")

var _ api.SectorManager = (*Manager)(nil)

var errMinerDisabled = fmt.Errorf("miner disblaed")

func NewManager(
	scfg *modules.SafeConfig,
	mapi api.MinerInfoAPI,
	numAlloc api.SectorNumberAllocator,
) (*Manager, error) {
	mgr := &Manager{
		msel: &minerSelector{
			scfg:  scfg,
			minfo: mapi,
		},
		numAlloc: numAlloc,
	}

	return mgr, nil
}

type Manager struct {
	msel     *minerSelector
	numAlloc api.SectorNumberAllocator
}

func (m *Manager) Allocate(ctx context.Context, allowedMiners []abi.ActorID, allowedProofs []abi.RegisteredSealProof) (*api.AllocatedSector, error) {
	candidates := m.msel.candidates(ctx, allowedMiners, allowedProofs)
	for {
		candidateCount := len(candidates)
		if candidateCount == 0 {
			return nil, nil
		}

		selectIdx := rand.Intn(candidateCount)
		selected := candidates[selectIdx]

		var check func(uint64) bool
		if selected.cfg.MaxNumber == nil {
			check = func(uint64) bool { return true }
		} else {
			max := *selected.cfg.MaxNumber
			check = func(next uint64) bool {
				ok := next <= max
				if !ok {
					log.Warnw("max number exceeded", "max", max, "miner", selected.info.ID)
				}
				return ok
			}
		}

		next, available, err := m.numAlloc.Next(ctx, selected.info.ID, selected.cfg.InitNumber, check)
		if err != nil {
			return nil, err
		}

		if available {
			return &api.AllocatedSector{
				ID: abi.SectorID{
					Miner:  selected.info.ID,
					Number: abi.SectorNumber(next),
				},
				ProofType: selected.info.SealProofType,
			}, nil
		}

		candidates[candidateCount-1], candidates[selectIdx] = candidates[selectIdx], candidates[candidateCount-1]
		candidates = candidates[:candidateCount-1]
	}
}
