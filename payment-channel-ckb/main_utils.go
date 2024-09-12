package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"time"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/perun-network/perun-libp2p-wire/p2p"
	"perun.network/go-perun/channel"
	"perun.network/go-perun/channel/persistence"
	"perun.network/go-perun/channel/persistence/keyvalue"
	"perun.network/perun-ckb-backend/channel/asset"
	"perun.network/perun-ckb-backend/wallet"
	"perun.network/perun-ckb-demo/client"
	"perun.network/perun-ckb-demo/deployment"
	"polycry.pt/poly-go/sortedkv/memorydb"
)

func printAllocationBalances(ch *client.PaymentChannel, asset asset.Asset, name string) {
	chAlloc := ch.State().Allocation
	idx := ch.GetParticipantIdx()
	// _assets := chAlloc.Assets
	// log.Println("Assets held by " + name)
	/*
		for _, a := range _assets {
			log.Println(a)
		}
	*/
	log.Println(name + "'s allocation in channel: " + chAlloc.Balance(idx, &asset).String())
}

func parseSUDTOwnerLockArg(pathToSUDTOwnerLockArg string) (string, error) {
	b, err := ioutil.ReadFile(pathToSUDTOwnerLockArg)
	if err != nil {
		return "", fmt.Errorf("reading sudt owner lock arg from file: %w", err)
	}
	sudtOwnerLockArg := string(b)
	if sudtOwnerLockArg == "" {
		return "", errors.New("sudt owner lock arg not found in file")
	}
	return sudtOwnerLockArg, nil
}

type PaymentClientSetup struct {
	Account         *wallet.Account
	Key             *secp256k1.PrivateKey
	PersistRestorer persistence.PersistRestorer
	WireAccount     *p2p.Account
	Net             *p2p.Net
}

func setupUser(name string, w *wallet.EphemeralWallet) (*PaymentClientSetup, error) {
	userKey, err := deployment.GetKey("./devnet/accounts/" + name + ".pk")
	if err != nil {
		log.Fatalf("error getting %s's private key: %v", name, err)
		return nil, err
	}

	userAccount := wallet.NewAccountFromPrivateKey(userKey)

	err = w.AddAccount(userAccount)
	if err != nil {
		log.Fatalf("error adding %s's account: %v", name, err)
		return nil, err
	}

	userWireAcc := p2p.NewRandomAccount(rand.New(rand.NewSource(time.Now().UnixNano())))
	userNet, err := p2p.NewP2PBus(userWireAcc)
	if err != nil {
		log.Fatalf("error creating p2p bus for %s: %v", name, err)
		return nil, err
	}
	userBus := userNet.Bus
	userListener := userNet.Listener
	go userBus.Listen(userListener)

	return &PaymentClientSetup{
		Account:         userAccount,
		Key:             userKey,
		PersistRestorer: keyvalue.NewPersistRestorer(memorydb.NewDatabase()),
		WireAccount:     userWireAcc,
		Net:             userNet,
	}, nil
}

func OpenLedgerChannel(proposer *client.PaymentClient, peer *client.PaymentClient, amounts map[channel.Asset]float64) (*client.PaymentChannel, *client.PaymentChannel) {
	// Open a ledger channel between proposer and responder
	ledgerChannelProposerToPeer := proposer.OpenChannel(peer.WireAddress(), peer.PeerID(), amounts)
	ledgerChannelPeerToProposer := peer.AcceptedChannel()

	log.Println("Ledger channel opened between " + proposer.Name + " and " + peer.Name)
	log.Println(proposer.Name, "'s balance ", proposer.GetBalances())
	log.Println(peer.Name, "'s balance ", peer.GetBalances())

	return ledgerChannelProposerToPeer, ledgerChannelPeerToProposer
}
