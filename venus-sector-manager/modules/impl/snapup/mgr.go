package snapup

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-bitfield"
	market7 "github.com/filecoin-project/specs-actors/v7/actors/builtin/market"

	"github.com/ipfs-force-community/venus-cluster/venus-sector-manager/api"
	"github.com/ipfs-force-community/venus-cluster/venus-sector-manager/pkg/chain"
	"github.com/ipfs-force-community/venus-cluster/venus-sector-manager/pkg/kvstore"
	// "github.com/ipfs-force-community/venus-cluster/venus-sector-manager/pkg/logging"
)

// var log = logging.New("snap")

type Manager struct {
	chain chain.API

	kvMu sync.Mutex
	kv   kvstore.KVStore
}

func (m *Manager) PreFetch(ctx context.Context, maddr address.Address, dlindex *uint64) (uint64, error) {
	ts, err := m.chain.ChainHead(ctx)
	if err != nil {
		return 0, fmt.Errorf("get chain head: %w", err)
	}

	tsk := ts.Key()
	tsh := ts.Height()

	deadline, err := m.chain.StateMinerProvingDeadline(ctx, maddr, tsk)
	if err != nil {
		return 0, fmt.Errorf("get proving deadline: %w", err)
	}

	didx := uint64(0)
	if dlindex != nil {
		didx = *dlindex % deadline.WPoStPeriodDeadlines
	} else {
		didx = deadline.Index
	}

	partitions, err := m.chain.StateMinerPartitions(ctx, maddr, didx, tsk)
	if err != nil {
		return 0, fmt.Errorf("get partitions: %w", err)
	}

	actives := bitfield.New()

	for pi, p := range partitions {
		psectors, err := m.chain.StateMinerSectors(ctx, maddr, &p.ActiveSectors, tsk)
		if err != nil {
			return 0, fmt.Errorf("get #%d partition info: %w", pi, err)
		}

		for _, sinfo := range psectors {
			if sinfo.SectorKeyCID == nil && len(sinfo.DealIDs) == 0 && sinfo.Expiration-tsh >= market7.DealMinDuration {
				actives.Set(uint64(sinfo.SectorNumber))
			}
		}
	}

	count, err := actives.Count()
	if err != nil {
		return 0, fmt.Errorf("get active count: %w", err)
	}

	if count == 0 {
		return 0, nil
	}

	diff, err := m.addSectors(ctx, maddr, actives, count)
	if err != nil {
		return 0, fmt.Errorf("add active sectors: %w", err)
	}

	return diff, nil
}

func (m *Manager) Allocate(ctx context.Context, spec api.AllocateSectorSpec) (*api.AllocatedSnapUpSector, error) {
	panic("not impl")
}

func (m *Manager) allocateForMiner(ctx context.Context, maddr address.Address) (*api.AllocatedSector, error) {
	panic("not impl")
}

func (m *Manager) addSectors(ctx context.Context, maddr address.Address, sectors bitfield.BitField, count uint64) (uint64, error) {
	m.kvMu.Lock()
	defer m.kvMu.Unlock()

	key := kvstore.Key(maddr.String())

	var exist *bitfield.BitField
	var existCount uint64
	err := m.kv.View(ctx, key, func(v kvstore.Val) error {
		var b bitfield.BitField
		err := json.Unmarshal(v, &b)
		if err != nil {
			return fmt.Errorf("bitfield from bytes: %w", err)
		}

		bcount, err := b.Count()
		if err != nil {
			return fmt.Errorf("get count from loaded bitfield: %w", err)
		}

		exist = &b
		existCount = bcount
		return nil
	})

	if err != nil {
		if !errors.Is(err, kvstore.ErrKeyNotFound) {
			return 0, fmt.Errorf("load exist sectors: %w", err)
		}
	}

	if existCount == 0 {
		exist = &sectors
	} else {
		merged, err := bitfield.MergeBitFields(*exist, sectors)
		if err != nil {
			return 0, fmt.Errorf("merge sectors: %w", err)
		}

		exist = &merged
	}

	mergedCount, err := exist.Count()
	if err != nil {
		return 0, fmt.Errorf("get merged count: %w", err)
	}

	if mergedCount == existCount {
		return 0, nil
	}

	data, err := json.Marshal(exist)
	if err != nil {
		return 0, fmt.Errorf("marshal merged bitfield: %w", err)
	}

	err = m.kv.Put(ctx, key, data)
	if err != nil {
		return 0, fmt.Errorf("save merged bitfied: %w", err)
	}

	return mergedCount - existCount, nil
}
