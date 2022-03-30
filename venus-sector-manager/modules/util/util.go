package util

import (
	"fmt"
	"path/filepath"

	commcid "github.com/filecoin-project/go-fil-commcid"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/ipfs/go-cid"
)

const (
	ss2KiB   = 2 << 10
	ss8MiB   = 8 << 20
	ss512MiB = 512 << 20
	ss32GiB  = 32 << 30
	ss64GiB  = 64 << 30
)

var ErrInvalidSectorSize = fmt.Errorf("invalid sector size")

func SectorSize2SealProofType(size uint64) (abi.RegisteredSealProof, error) {
	switch size {
	case ss2KiB:
		return abi.RegisteredSealProof_StackedDrg2KiBV1_1, nil

	case ss8MiB:
		return abi.RegisteredSealProof_StackedDrg8MiBV1_1, nil

	case ss512MiB:
		return abi.RegisteredSealProof_StackedDrg512MiBV1_1, nil

	case ss32GiB:
		return abi.RegisteredSealProof_StackedDrg32GiBV1_1, nil

	case ss64GiB:
		return abi.RegisteredSealProof_StackedDrg64GiBV1_1, nil

	default:
		return 0, fmt.Errorf("%w: %d", ErrInvalidSectorSize, size)
	}
}

type pathType string

const (
	SectorPathTypeCache    = "cache"
	SectorPathTypeSealed   = "sealed"
	SectorPathTypeUnsealed = "unsealed"
)

const sectorIDFormat = "s-t0%d-%d"

func SectorPath(typ pathType, sid abi.SectorID) string {
	return filepath.Join(string(typ), FormatSectorID(sid))
}

func FormatSectorID(sid abi.SectorID) string {
	return fmt.Sprintf(sectorIDFormat, sid.Miner, sid.Number)
}

func ScanSectorID(s string) (abi.SectorID, bool) {
	var sid abi.SectorID
	read, err := fmt.Sscanf(s, sectorIDFormat, &sid.Miner, &sid.Number)
	return sid, err == nil && read == 2
}

func ReplicaCommitment2CID(commR [32]byte) (cid.Cid, error) {
	return commcid.ReplicaCommitmentV1ToCID(commR[:])
}

func CID2ReplicaCommitment(sealedCID cid.Cid) ([32]byte, error) {
	var commR [32]byte

	if !sealedCID.Defined() {
		return commR, fmt.Errorf("undefined cid")
	}

	b, err := commcid.CIDToReplicaCommitmentV1(sealedCID)
	if err != nil {
		return commR, fmt.Errorf("convert to commitment: %w", err)
	}

	if size := len(b); size != 32 {
		return commR, fmt.Errorf("get %d bytes for commitment", size)
	}

	copy(commR[:], b[:])
	return commR, nil
}
