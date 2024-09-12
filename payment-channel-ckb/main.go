package main

import (
	"log"
	"os"

	"github.com/nervosnetwork/ckb-sdk-go/v2/types"
	"perun.network/go-perun/channel"
	"perun.network/perun-ckb-backend/channel/asset"
	"perun.network/perun-ckb-backend/wallet"
	"perun.network/perun-ckb-demo/client"
	"perun.network/perun-ckb-demo/deployment"
)

const (
	rpcNodeURL = "http://localhost:8114"
	Network    = types.NetworkTest
)

func SetLogFile(path string) {
	logFile, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	log.SetOutput(logFile)
}

func main() {
	SetLogFile("demo.log")
	log.Println("\n\nStarting demo")
	sudtOwnerLockArg, err := parseSUDTOwnerLockArg("./devnet/accounts/sudt-owner-lock-hash.txt")
	if err != nil {
		log.Fatalf("error getting SUDT owner lock arg: %v", err)
	}
	d, _, err := deployment.GetDeployment("./devnet/contracts/migrations/dev/", "./devnet/system_scripts", sudtOwnerLockArg)
	log.Printf("deployment object: %v", d)
	if err != nil {
		log.Fatalf("error getting deployment: %v", err)
	}
	/*
		maxSudtCapacity := transaction.CalculateCellCapacity(types.CellOutput{
			Capacity: 0,
			Lock:     &d.DefaultLockScript,
			Type:     sudtInfo.Script,
		})
	*/
	w := wallet.NewEphemeralWallet()

	alicePaymentClientSetup, err := setupUser("alice", w)
	if err != nil {
		log.Fatalf("error setting up Alice's client: %v", err)
	}
	alice, err := client.NewPaymentClient(
		"alice",
		Network,
		d,
		rpcNodeURL,
		alicePaymentClientSetup.Account,
		*alicePaymentClientSetup.Key,
		w,
		alicePaymentClientSetup.PersistRestorer,
		alicePaymentClientSetup.WireAccount.Address(),
		alicePaymentClientSetup.Net,
	)
	if err != nil {
		log.Fatalf("error setting up Alice's client: %v", err)
	}
	// alice, err := setupPaymentClientForUser("alice", w, d, prAlice)
	// if err != nil {
	// 	log.Fatalf("error setting up Alice's client: %v", err)
	// }

	bobPaymentClientSetup, err := setupUser("bob", w)
	if err != nil {
		log.Fatalf("error setting up Bob's client: %v", err)
	}
	bob, err := client.NewPaymentClient(
		"bob",
		Network,
		d,
		rpcNodeURL,
		bobPaymentClientSetup.Account,
		*bobPaymentClientSetup.Key,
		w,
		bobPaymentClientSetup.PersistRestorer,
		bobPaymentClientSetup.WireAccount.Address(),
		bobPaymentClientSetup.Net,
	)
	if err != nil {
		log.Fatalf("error setting up Bob's client: %v", err)
	}

	// ingrid, err := setupPaymentClientForUser("ingrid", w, d, prIngrid)
	// if err != nil {
	// 	log.Fatalf("error setting up Ingrid's client: %v", err)
	// }
	ingridPaymentClientSetup, err := setupUser("ingrid", w)
	if err != nil {
		log.Fatalf("error setting up Ingrid's client: %v", err)
	}
	ingrid, err := client.NewPaymentClient(
		"ingrid",
		Network,
		d,
		rpcNodeURL,
		ingridPaymentClientSetup.Account,
		*ingridPaymentClientSetup.Key,
		w,
		ingridPaymentClientSetup.PersistRestorer,
		ingridPaymentClientSetup.WireAccount.Address(),
		ingridPaymentClientSetup.Net,
	)
	if err != nil {
		log.Fatalf("error setting up Ingrid's client: %v", err)
	}

	log.Println("Balances of Alice and Bob before transaction")
	str := "'s account balance"
	log.Println(alice.Name, str, alice.GetBalances())
	log.Println(bob.Name, str, bob.GetBalances())
	log.Println(ingrid.Name, str, ingrid.GetBalances())

	ckbAsset := asset.Asset{
		IsCKBytes: true,
		SUDT:      nil,
	}

	/*
		sudtAsset := asset.Asset{
			IsCKBytes: false,
			SUDT: &asset.SUDT{
				TypeScript:  *sudtInfo.Script,
				MaxCapacity: maxSudtCapacity,
			},
		}
	*/

	log.Println("Opening channel between Alice and Ingrid")
	// chAliceIngrid := alice.OpenChannel(ingrid.WireAddress(), ingrid.PeerID(), map[channel.Asset]float64{
	// 	&ckbAsset: 100.0,
	// })
	// chIngridAlice := ingrid.AcceptedChannel()
	// log.Println(alice.Name, str, alice.GetBalances())
	// log.Println(ingrid.Name, str, ingrid.GetBalances())
	// log.Println("Sending payments....")
	chAliceIngrid, chIngridAlice := OpenLedgerChannel(alice, ingrid, map[channel.Asset]float64{
		&ckbAsset: 100.0,
	})

	chAliceIngrid.SendPayment(map[channel.Asset]float64{&ckbAsset: 10.0})

	// log.Println("Alice sent Ingrid a payment")
	// printAllocationBalances(chAliceIngrid, ckbAsset, "Alice")
	// printAllocationBalances(chIngridAlice, ckbAsset, "Ingrid")

	chIngridAlice.SendPayment(map[channel.Asset]float64{
		&ckbAsset: 10.0,
	})
	// log.Println("Ingrid sent Alice a payment")
	// printAllocationBalances(chAliceIngrid, ckbAsset, "Alice")
	// printAllocationBalances(chIngridAlice, ckbAsset, "Ingrid")

	log.Println("Opening channel Ingrid and Bob")
	chIngridBob, chBobIngrid := OpenLedgerChannel(ingrid, bob, map[channel.Asset]float64{
		&ckbAsset: 100.0,
	})
	// chIngridBob := ingrid.OpenChannel(ingrid.WireAddress(), ingrid.PeerID(), map[channel.Asset]float64{
	// 	&ckbAsset: 100.0,
	// })
	// log.Println(ingrid.Name, str, ingrid.GetBalances())
	// log.Println(bob.Name, str, bob.GetBalances())
	// chIngrid := bob.AcceptedChannel()

	log.Println("Sending payments....")
	chIngridBob.SendPayment(map[channel.Asset]float64{
		&ckbAsset: 10.0,
	})
	// log.Println("Ingrid sent Bob a payment")
	// printAllocationBalances(chIngridBob, ckbAsset, "Ingrid")
	// printAllocationBalances(chBobIngrid, ckbAsset, "Bob")

	chBobIngrid.SendPayment(map[channel.Asset]float64{
		&ckbAsset: 10.0,
	})
	// log.Println("Bob sent Ingrid a payment")
	// printAllocationBalances(chIngridBob, ckbAsset, "ingrid")
	// printAllocationBalances(chBobIngrid, ckbAsset, "Ingrid")

	log.Println("Demo Successful Exit")

}
