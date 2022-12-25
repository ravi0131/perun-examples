# multi-ledger-channel

Install [ganache-cli](https://github.com/trufflesuite/ganache-cli) and start the
first chain by running:
```sh
KEY_DEPLOYER=0x79ea8f62d97bc0591a4224c1725fca6b00de5b2cea286fe2e0bb35c5e76be46e
KEY_ALICE=0x1af2e950272dd403de7a5760d41c6e44d92b6d02797e51810795ff03cc2cda4f
KEY_BOB=0xf63d7d8e930bccd74e93cf5662fde2c28fd8be95edb70c73f1bdd863d07f412e
BALANCE=100000000000000000000000

ganache-cli --host 127.0.0.1 --port 8545 --chain.chainId 1337 --account $KEY_DEPLOYER,$BALANCE --account $KEY_ALICE,$BALANCE --account $KEY_BOB,$BALANCE --gasPrice=0
```
Open up a second terminal and start the second chain:
```sh
KEY_DEPLOYER=0x79ea8f62d97bc0591a4224c1725fca6b00de5b2cea286fe2e0bb35c5e76be46e
KEY_ALICE=0x1af2e950272dd403de7a5760d41c6e44d92b6d02797e51810795ff03cc2cda4f
KEY_BOB=0xf63d7d8e930bccd74e93cf5662fde2c28fd8be95edb70c73f1bdd863d07f412e
BALANCE=100000000000000000000000

ganache-cli --host 127.0.0.1 --port 8546 --chain.chainId 1338 --account $KEY_DEPLOYER,$BALANCE --account $KEY_ALICE,$BALANCE --account $KEY_BOB,$BALANCE --gasPrice=0
```


Then run
```
go run .
```