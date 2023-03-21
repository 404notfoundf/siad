package pool

import (
	"gitlab.com/NebulousLabs/encoding"
	"go.sia.tech/siad/types"
	"io"
)

type GlobalConfig struct {
	MiningPool struct {
		Name       string `yaml:"name"`
		PoolWallet string `yaml:"poolwallet"`
		PoolWebUrl string `yaml:"poolweburl"`
		PoolLogDir string `yaml:"poollogdir"`
	}
}

type PoolConfig struct {
	PoolName   string
	PoolWallet string
	PoolWebUrl string
	PoolLogDir string
}

type ConsensusNotify struct {
	Target    types.Target      `json:"target"`
	Height    types.BlockHeight `json:"height"`
	Block     types.Block       `json:"block"`
	Coinbase1 string            `json:"coinbase_1"`
	Coinbase2 string            `json:"coinbase_2"`
	Merkle    []string          `json:"merkle"`
	Nbits     string            `json:"nbits"`
	Ntime     string            `json:"ntime"`
	Error     string            `json:"error"`
}

type SubmitResponse struct {
	Message string `json:"message"`
}

func MarshalSiaNoSignatures(t types.Transaction, w io.Writer) {
	e := encoding.NewEncoder(w)
	e.WriteInt(len((t.SiacoinInputs)))
	for i := range t.SiacoinInputs {
		t.SiacoinInputs[i].MarshalSia(e)
	}
	e.WriteInt(len((t.SiacoinOutputs)))
	for i := range t.SiacoinOutputs {
		t.SiacoinOutputs[i].MarshalSia(e)
	}
	e.WriteInt(len((t.FileContracts)))
	for i := range t.FileContracts {
		t.FileContracts[i].MarshalSia(e)
	}
	e.WriteInt(len((t.FileContractRevisions)))
	for i := range t.FileContractRevisions {
		t.FileContractRevisions[i].MarshalSia(e)
	}
	e.WriteInt(len((t.StorageProofs)))
	for i := range t.StorageProofs {
		t.StorageProofs[i].MarshalSia(e)
	}
	e.WriteInt(len((t.SiafundInputs)))
	for i := range t.SiafundInputs {
		t.SiafundInputs[i].MarshalSia(e)
	}
	e.WriteInt(len((t.SiafundOutputs)))
	for i := range t.SiafundOutputs {
		t.SiafundOutputs[i].MarshalSia(e)
	}
	e.WriteInt(len((t.MinerFees)))
	for i := range t.MinerFees {
		t.MinerFees[i].MarshalSia(e)
	}
	e.WriteInt(len((t.ArbitraryData)))
	for i := range t.ArbitraryData {
		e.WritePrefixedBytes(t.ArbitraryData[i])
	}
}
