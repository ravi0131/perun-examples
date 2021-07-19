// Copyright 2021 PolyCrypt GmbH, Germany
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"crypto/ecdsa"
	"fmt"
	"log"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
	"perun.network/perun-examples/app-channel/client"
	"perun.network/perun-examples/app-channel/eth"
)

func logAccountBalance(clients ...*client.Client) {
	for _, c := range clients {
		globalBalance, err := c.OnChainBalance()
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("%v: Account Balance: %v", c.Address(), toEth(globalBalance))
	}
}

func toEth(weiAmount *big.Int) string {
	return fmt.Sprintf("%vETH", eth.WeiToEth(weiAmount))
}

func deployContracts(nodeURL string, chainID *big.Int, deploymentKey *ecdsa.PrivateKey, contextTimeout time.Duration, collateralWithdrawalDelay time.Duration) (contracts ContractAddresses, err error) {
	ethClient, err := eth.NewEthClient(nodeURL, deploymentKey, chainID, contextTimeout)
	if err != nil {
		err = errors.WithMessage(err, "creating ethereum client")
		return
	}

	// Deploy adjudicator
	adjudicatorAddr, txAdj, err := ethClient.DeployAdjudicator()
	if err != nil {
		err = errors.WithMessage(err, "deploying adjudicator")
		return
	}

	// Deploy collateralized channels app
	appAddr, txApp, err := ethClient.DeployApp(adjudicatorAddr)
	if err != nil {
		err = errors.WithMessage(err, "deploying CollateralApp")
		return
	}

	// Deploy asset holder
	assetHolderAddr, txAss, err := ethClient.DeployAssetHolderETH(adjudicatorAddr, appAddr, big.NewInt(int64(collateralWithdrawalDelay.Seconds())))
	if err != nil {
		err = errors.WithMessage(err, "deploying CollateralAssetHolderETH")
		return
	}

	err = ethClient.WaitDeployment(txAdj, txApp, txAss)
	if err != nil {
		err = errors.WithMessage(err, "waiting for contract deployment")
		return
	}

	return ContractAddresses{
		AdjudicatorAddr: adjudicatorAddr,
		AssetHolderAddr: assetHolderAddr,
		AppAddr:         appAddr,
	}, nil
}

type ContractAddresses struct {
	AdjudicatorAddr, AssetHolderAddr, AppAddr common.Address
}
