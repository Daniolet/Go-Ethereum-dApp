package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/jjg-akers/go-ethereum-dapp/quiz"
	"github.com/joho/godotenv"
)

var myenv map[string]string

const envLoc = ".env"

func loadEnv() {
	var err error
	if err = godotenv.Load(envLoc); err != nil {
		log.Printf("could not load env from %s: %v", envLoc, err)
	}
}

//This will initialize a connection to the Rinkeby network, using the infura.io gateway endpoint
func main() {
	loadEnv()

	// fmt.Println("gateway: ", os.Getenv("GATEWAY"))
	// fmt.Println("pubkey: ", os.Getenv("PUBLICKEY"))

	ctx := context.Background()

	client, err := ethclient.Dial(os.Getenv("GATEWAY"))
	if err != nil {
		log.Fatalf("could not connect to Ethereum gateway: %v\n", err)
	}

	defer client.Close()

	// Need to convert our address from a hex string to common.Address type
	accountAddress := common.HexToAddress(os.Getenv("PUBLICKEY"))

	// read the balance at our address
	balance, _ := client.BalanceAt(ctx, accountAddress, nil)
	fmt.Printf("Balance: %d\n", balance)
}

//Sessions are wrappers that allow us to make contract calls without having to pass aorund auth, creds,
//  and call parameters constantly
//A session wraps:
//  a contract instance
//	a bind.CallOpts struct that contains opts for making contract calls
//	a bind.TransactOpts struct that contains auth creds and params for created a valid Ethereum transaction

func NewSession(ctx context.Context) (session quiz.QuizSession) {
	loadEnv()
	keystore, err := os.Open(os.Getenv("KEYSTORE"))
	if err != nil {
		log.Printf("could not load keystore: %s\n", err)
	}

	defer keystore.Close()

	keystorepass := os.Getenv("KEYSTOREPASS")

	// get transact opts
	auth, err := bind.NewTransactor(keystore, keystorepass)
	if err != nil {
		log.Println("Error getting auth: ", err)
	}

	callOpts := bind.CallOpts{
		From:    auth.From,
		Context: ctx,
	}

	//return session without contract instance
	return quiz.QuizSession{
		TransactOpts: *auth,
		CallOpts:     callOpts,
	}
}
