// Copyright 2018 The dexon-consensus-core Authors
// This file is part of the dexon-consensus-core library.
//
// The dexon-consensus-core library is free software: you can redistribute it
// and/or modify it under the terms of the GNU Lesser General Public License as
// published by the Free Software Foundation, either version 3 of the License,
// or (at your option) any later version.
//
// The dexon-consensus-core library is distributed in the hope that it will be
// useful, but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU Lesser
// General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the dexon-consensus-core library. If not, see
// <http://www.gnu.org/licenses/>.

package test

import (
	"testing"
	"time"

	"github.com/dexon-foundation/dexon-consensus-core/common"
	"github.com/dexon-foundation/dexon-consensus-core/core"
	"github.com/dexon-foundation/dexon-consensus-core/core/blockdb"
	"github.com/dexon-foundation/dexon-consensus-core/core/types"
	"github.com/stretchr/testify/suite"
)

type StopperTestSuite struct {
	suite.Suite
}

func (s *StopperTestSuite) TestStopByConfirmedBlocks() {
	// This test case makes sure this stopper would stop when
	// all nodes confirmed at least 'x' count of blocks produced
	// by themselves.
	var (
		req = s.Require()
	)

	apps := make(map[types.NodeID]*App)
	dbs := make(map[types.NodeID]blockdb.BlockDatabase)
	nodes := GenerateRandomNodeIDs(2)
	db, err := blockdb.NewMemBackedBlockDB()
	req.Nil(err)
	for _, nID := range nodes {
		apps[nID] = NewApp()
		dbs[nID] = db
	}
	deliver := func(blocks []*types.Block) {
		hashes := common.Hashes{}
		for _, b := range blocks {
			hashes = append(hashes, b.Hash)
			req.Nil(db.Put(*b))
		}
		for _, nID := range nodes {
			app := apps[nID]
			for _, h := range hashes {
				app.StronglyAcked(h)
			}
			app.TotalOrderingDelivered(hashes, core.TotalOrderingModeNormal)
			for _, h := range hashes {
				app.BlockDelivered(h, types.FinalizationResult{
					Timestamp: time.Time{},
				})
			}
		}
	}
	stopper := NewStopByConfirmedBlocks(2, apps, dbs)
	b00 := &types.Block{
		ProposerID: nodes[0],
		Hash:       common.NewRandomHash(),
	}
	deliver([]*types.Block{b00})
	b10 := &types.Block{
		ProposerID: nodes[1],
		Hash:       common.NewRandomHash(),
	}
	b11 := &types.Block{
		ProposerID: nodes[1],
		ParentHash: b10.Hash,
		Hash:       common.NewRandomHash(),
	}
	deliver([]*types.Block{b10, b11})
	req.False(stopper.ShouldStop(nodes[1]))
	b12 := &types.Block{
		ProposerID: nodes[1],
		ParentHash: b11.Hash,
		Hash:       common.NewRandomHash(),
	}
	deliver([]*types.Block{b12})
	req.False(stopper.ShouldStop(nodes[1]))
	b01 := &types.Block{
		ProposerID: nodes[0],
		ParentHash: b00.Hash,
		Hash:       common.NewRandomHash(),
	}
	deliver([]*types.Block{b01})
	req.True(stopper.ShouldStop(nodes[0]))
}

func TestStopper(t *testing.T) {
	suite.Run(t, new(StopperTestSuite))
}
