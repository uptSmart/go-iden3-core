package mock

import (
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/iden3/go-iden3-core/core"
	"github.com/iden3/go-iden3-core/core/proof"
	"github.com/iden3/go-iden3-core/merkletree"
	"github.com/stretchr/testify/mock"
)

type IdenStateReadMock struct {
	mock.Mock
}

func New() *IdenStateReadMock {
	return &IdenStateReadMock{}
}

func (m *IdenStateReadMock) GetState(id *core.ID) (*proof.IdenStateData, error) {
	args := m.Called(id)
	return args.Get(0).(*proof.IdenStateData), args.Error(1)
}

func (m *IdenStateReadMock) GetStateByBlock(id *core.ID, blockN uint64) (merkletree.Hash, error) {
	args := m.Called(id, blockN)
	return args.Get(0).(merkletree.Hash), args.Error(1)
}

func (m *IdenStateReadMock) GetStateByTime(id *core.ID, blockTimeStamp int64) (merkletree.Hash, error) {
	args := m.Called(id, blockTimeStamp)
	return args.Get(0).(merkletree.Hash), args.Error(1)
}

func (m *IdenStateReadMock) SetState(id *core.ID, newState *merkletree.Hash, kOpProof []byte, stateTransitionProof []byte, signature *merkletree.Hash) (*types.Transaction, error) {
	args := m.Called(id, newState, kOpProof, stateTransitionProof, signature)
	return args.Get(0).(*types.Transaction), args.Error(1)
}

// func (m *IdenStateReadMock) VerifyProofClaim(pc *proof.ProofClaim) (bool, error) {
// 	args := m.Called(pc)
// 	return args.Get(0).(bool), args.Error(1)
// }

// func (m *IdenStateReadMock) Client() *eth.Client2 {
// 	args := m.Called()
// 	return args.Get(0).(*eth.Client2)
// }
