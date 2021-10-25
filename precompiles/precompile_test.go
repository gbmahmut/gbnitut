//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package precompiles

import (
	"bytes"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/offchainlabs/arbstate/arbos/storage"
	templates "github.com/offchainlabs/arbstate/solgen/go"
	"math/big"
	"testing"
)

func TestEvents(t *testing.T) {

	blockNumber := 1024

	// create a minimal evm that supports just enough to create logs
	evm := vm.EVM{
		StateDB: storage.NewMemoryBackedStateDB(),
		Context: vm.BlockContext{
			BlockNumber: big.NewInt(int64(blockNumber)),
			GasLimit:    ^uint64(0),
		},
	}

	debugContractAddr := common.HexToAddress("ff")
	contract := Precompiles()[debugContractAddr]

	var method PrecompileMethod

	for _, available := range contract.Precompile().methods {
		if available.name == "Events" {
			method = available
			break
		}
	}

	zeroHash := crypto.Keccak256([]byte{0x00})
	trueHash := common.Hash{}.Bytes()
	falseHash := common.Hash{}.Bytes()
	trueHash[31] = 0x01

	var data []byte
	payload := [][]byte{
		method.template.ID, // select the `Events` method
		falseHash,          // set the flag to false
		zeroHash,           // set the value to something known
	}
	for _, bytes := range payload {
		data = append(data, bytes...)
	}

	caller := common.HexToAddress("aaaaaaaabbbbbbbbccccccccdddddddd")
	number := big.NewInt(0x9364)

	output, err := contract.Call(
		data,
		debugContractAddr,
		debugContractAddr,
		caller,
		number,
		false,
		&evm,
	)
	check(t, err, "call failed")

	outputAddr := common.BytesToAddress(output[:32])
	outputData := new(big.Int).SetBytes(output[32:])

	if outputAddr != caller {
		t.Fatal("unexpected output address", outputAddr, "instead of", caller)
	}
	if outputData.Cmp(number) != 0 {
		t.Fatal("unexpected output number", outputData, "instead of", number)
	}

	//nolint:errcheck
	logs := evm.StateDB.(*state.StateDB).Logs()
	for _, log := range logs {
		if log.Address != debugContractAddr {
			t.Fatal("address mismatch:", log.Address, "vs", debugContractAddr)
		}
		if log.BlockNumber != uint64(blockNumber) {
			t.Fatal("block number mismatch:", log.BlockNumber, "vs", blockNumber)
		}
		t.Log("topic", len(log.Topics), log.Topics)
		t.Log("data ", len(log.Data), log.Data)
	}

	basicTopics := logs[0].Topics
	mixedTopics := logs[1].Topics

	if !bytes.Equal(basicTopics[1].Bytes(), zeroHash) || !bytes.Equal(mixedTopics[2].Bytes(), zeroHash) {
		t.Fatal("indexing a bytes32 didn't work")
	}
	if !bytes.Equal(mixedTopics[1].Bytes(), falseHash) {
		t.Fatal("indexing a bool didn't work")
	}
	if !bytes.Equal(mixedTopics[3].Bytes(), caller.Hash().Bytes()) {
		t.Fatal("indexing an address didn't work")
	}

	ArbDebugInfo, cerr := templates.NewArbDebug(common.Address{}, nil)
	basic, berr := ArbDebugInfo.ParseBasic(*logs[0])
	mixed, merr := ArbDebugInfo.ParseMixed(*logs[1])
	if cerr != nil || berr != nil || merr != nil {
		t.Fatal("failed to parse event logs", "\nprecompile:", cerr, "\nbasic:", berr, "\nmixed:", merr)
	}

	if basic.Flag != true || !bytes.Equal(basic.Value[:], zeroHash) {
		t.Fatal("event Basic's data isn't correct")
	}
	if mixed.Flag != false || mixed.Not != true || !bytes.Equal(mixed.Value[:], zeroHash) {
		t.Fatal("event Mixed's data isn't correct")
	}
	if mixed.Conn != debugContractAddr || mixed.Caller != caller {
		t.Fatal("event Mixed's data isn't correct")
	}
}

func check(t *testing.T, err error, str ...string) {
	if err != nil {
		t.Fatal(err, str)
	}
}