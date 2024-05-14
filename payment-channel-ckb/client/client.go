package client

import (
	"context"
	"fmt"
	"log"
	"math/big"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/nervosnetwork/ckb-sdk-go/v2/rpc"
	"github.com/nervosnetwork/ckb-sdk-go/v2/types"
	"perun.network/go-perun/channel"
	gpchannel "perun.network/go-perun/channel"
	"perun.network/go-perun/channel/persistence"
	"perun.network/go-perun/client"
	gpwallet "perun.network/go-perun/wallet"
	"perun.network/go-perun/watcher/local"
	"perun.network/go-perun/wire"
	"perun.network/go-perun/wire/net/simple"
	"perun.network/perun-ckb-backend/backend"
	"perun.network/perun-ckb-backend/channel/adjudicator"
	"perun.network/perun-ckb-backend/channel/asset"
	"perun.network/perun-ckb-backend/channel/funder"
	ckbclient "perun.network/perun-ckb-backend/client"
	"perun.network/perun-ckb-backend/wallet"
	"perun.network/perun-ckb-backend/wallet/address"
	"polycry.pt/poly-go/sync"
)

type PaymentClient struct {
	balanceMutex sync.Mutex
	Name         string
	balance      *big.Int
	sudtBalance  *big.Int
	Account      *wallet.Account
	wAddr        wire.Address
	Network      types.Network
	PerunClient  *client.Client

	channels  chan *PaymentChannel
	rpcClient rpc.Client
}

func NewPaymentClient(
	name string,
	network types.Network,
	deployment backend.Deployment,
	bus wire.Bus,
	rpcUrl string,
	account *wallet.Account,
	key secp256k1.PrivateKey,
	wallet *wallet.EphemeralWallet,
	persistRestorer persistence.PersistRestorer,
) (*PaymentClient, error) {
	backendRPCClient, err := rpc.Dial(rpcUrl)
	if err != nil {
		return nil, err
	}
	signer := backend.NewSignerInstance(address.AsParticipant(account.Address()).ToCKBAddress(network), key, network)

	ckbClient, err := ckbclient.NewClient(backendRPCClient, *signer, deployment)
	if err != nil {
		return nil, err
	}
	f := funder.NewDefaultFunder(ckbClient, deployment)
	a := adjudicator.NewAdjudicator(ckbClient)
	watcher, err := local.NewWatcher(a)
	if err != nil {
		return nil, err
	}
	wAddr := simple.NewAddress(account.Address().String())
	perunClient, err := client.New(wAddr, bus, f, a, wallet, watcher)
	if err != nil {
		return nil, err
	}
	perunClient.EnablePersistence(persistRestorer)

	balanceRPC, err := rpc.Dial(rpcUrl)
	if err != nil {
		return nil, err
	}
	p := &PaymentClient{
		Name:        name,
		balance:     big.NewInt(0),
		sudtBalance: big.NewInt(0),
		Account:     account,
		wAddr:       wAddr,
		Network:     network,
		PerunClient: perunClient,
		channels:    make(chan *PaymentChannel, 1),
		rpcClient:   balanceRPC,
	}

	//go p.PollBalances()
	go perunClient.Handle(p, p)
	return p, nil
}

// WalletAddress returns the wallet address of the client.
func (p *PaymentClient) WalletAddress() gpwallet.Address {
	return p.Account.Address()
}

func (p *PaymentClient) WireAddress() wire.Address {
	return p.wAddr
}

func (p *PaymentClient) GetSudtBalance() *big.Int {
	p.balanceMutex.Lock()
	defer p.balanceMutex.Unlock()
	return new(big.Int).Set(p.sudtBalance)
}

// TODO: Remove as probably not required
/*
func (p *PaymentClient) NotifyAllBalance(ckbBal int64) string {
	str := FormatBalance(new(big.Int).SetInt64(ckbBal), p.GetSudtBalance())
	return str
}
*/

// GetBalances retrieves the current balances of the client.
func (p *PaymentClient) GetBalances() string {
	p.PollBalances()
	return FormatBalance(p.balance, p.sudtBalance)
}

// OpenChannel opens a new channel with the specified peer and funding.
func (p *PaymentClient) OpenChannel(peer wire.Address, amounts map[gpchannel.Asset]float64) *PaymentChannel {
	// We define the channel participants. The proposer always has index 0. Here
	// we use the on-chain addresses as off-chain addresses, but we could also
	// use different ones.
	log.Println("OpenChannel called")
	participants := []wire.Address{p.WireAddress(), peer}

	assets := make([]gpchannel.Asset, len(amounts))
	i := 0
	for a := range amounts {
		assets[i] = a
		i++
	}

	// We create an initial allocation which defines the starting balances.
	initAlloc := gpchannel.NewAllocation(2, assets...)
	log.Println(initAlloc.Assets)
	for a, amount := range amounts {
		switch a := a.(type) {
		case *asset.Asset:
			if a.IsCKBytes {
				initAlloc.SetAssetBalances(a, []gpchannel.Bal{
					CKByteToShannon(big.NewFloat(amount)), // Our initial balance.
					CKByteToShannon(big.NewFloat(amount)), // Peer's initial balance.
				})
			} else {
				intAmount := new(big.Int).SetUint64(uint64(amount))
				initAlloc.SetAssetBalances(a, []gpchannel.Bal{
					intAmount, // Our initial balance.
					intAmount, // Peer's initial balance.
				})
			}
		default:
			panic("Asset is not of type *asset.Asset")
		}

	}
	log.Println("Created Allocation")

	// Prepare the channel proposal by defining the channel parameters.
	challengeDuration := uint64(10) // On-chain challenge duration in seconds.
	proposal, err := client.NewLedgerChannelProposal(
		challengeDuration,
		p.Account.Address(),
		initAlloc,
		participants,
	)
	if err != nil {
		panic(err)
	}

	log.Println("Created Proposal")

	// Send the proposal.
	ch, err := p.PerunClient.ProposeChannel(context.TODO(), proposal)
	if err != nil {
		panic(err)
	}

	log.Println("Sent Channel")

	// Start the on-chain event watcher. It automatically handles disputes.
	p.startWatching(ch)

	log.Println("Started Watching")

	//p.Channel = newPaymentChannel(ch, assets)
	return newPaymentChannel(ch, assets)
}

// startWatching starts the dispute watcher for the specified channel.
func (p *PaymentClient) startWatching(ch *client.Channel) {
	go func() {
		err := ch.Watch(p)
		if err != nil {
			fmt.Printf("Watcher returned with error: %v", err)
		}
	}()
}

func (p *PaymentClient) AcceptedChannel() *PaymentChannel {
	return <-p.channels
}

func (p *PaymentClient) Shutdown() {
	p.PerunClient.Close()
}

func (c *PaymentClient) Restore() []*PaymentChannel {
	var restoredChannels []*client.Channel

	//TODO: Remove this hack. Find why asset is not found upon restoring
	c.PerunClient.OnNewChannel(func(ch *client.Channel) {
		/*
			state := ch.State().Clone()
			ckbyte := asset.Asset{
				IsCKBytes: true,
				SUDT:      nil,
			}
			//create a new allocation where asset type is defined
			alloc := gpchannel.NewAllocation(2, ckbyte)
			ckbBalances := state.Allocation.Balances[0]
			alloc.SetAssetBalances(ckbyte, ckbBalances)
		*/
		restoredChannels = append(restoredChannels, ch)
	})

	err := c.PerunClient.Restore(context.TODO())
	if err != nil {
		fmt.Println("Error restoring channels")
	}

	paymentChannels := make([]*PaymentChannel, len(restoredChannels))
	assets := make([]channel.Asset, 1)
	assets = append(assets, &asset.Asset{
		IsCKBytes: true,
		SUDT:      nil,
	})
	for i, ch := range restoredChannels {
		paymentChannels[i] = newPaymentChannel(ch, assets)
	}

	return paymentChannels
}