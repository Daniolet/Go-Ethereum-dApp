package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/jjg-akers/go-ethereum-dapp/quiz"
	"github.com/jjg-akers/go-ethereum-dapp/utils"

	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/joho/godotenv"
)

var myenv map[string]string

const envLoc = ".env"

func loadEnv() {
	var err error
	if err = godotenv.Load(envLoc); err != nil {
		log.Printf("could not load env from %s: %v", envLoc, err)
	}

	if myenv, err = godotenv.Read(envLoc); err != nil {
		log.Printf("could not read in env from %s: %s", envLoc, err)
	}
}

// What we will do with this contract
//	- read the question
//	- send an answer to the contract
//	- check if answer sent is correct
//	- if answer is correct, record the user's account addr

//This will initialize a connection to the Rinkeby network, using the infura.io gateway endpoint
func main() {

	loadEnv()

	// fmt.Println("gateway: ", os.Getenv("GATEWAY"))
	// fmt.Println("pubkey: ", os.Getenv("PUBLICKEY"))

	ctx := context.Background()

	client, err := ethclient.Dial(os.Getenv("GATEWAY")) //address of testnet
	if err != nil {
		log.Fatalf("could not connect to Ethereum gateway: %v\n", err)
	}

	defer client.Close()

	// Need to convert our address from a hex string to common.Address type
	// accountAddress := common.HexToAddress(os.Getenv("PUBLICKEY"))

	// // read the balance at our address
	// balance, _ := client.BalanceAt(ctx, accountAddress, nil)
	// fmt.Printf("Balance: %d\n", balance)

	// create a new session
	// We havn't spedified a value for the contract field yet, we will do that after obtaining a contract instance
	// when we deploy a new contract or whaen we load an existing
	session := NewSession(ctx)

	// now we need to assigne it to a contrance instance - an instance is is obtained by deploying a contrace or loading and existing

	//load or deploy contract, and update session with contract instance
	if myenv["CONTRACTADDR"] == "" {
		session = NewContract(session, client, myenv["QUESTION"], myenv["ANSWER"])
	}

	// --- Deploy contract if doesn't exist
	// if we have an existing contract, load it; if we've deployed a new contract, attemp to load it.
	if myenv["CONTRACTADDR"] != "" {
		session = LoadContract(session, client)
	}
	//NOTE: To force the DApp to deploy a new contract, remove the CONTRACTADD entry in the .env file, or set it to an empty string

	// --- Interact with the Contract
	// now that we have an instance, we can use it to make calls -- we can access any public functions or variables on the the contract instance

	//CLI for running App - the cli should allow users to...
	//1. Read the question
	//2. Send an answer
	//3. Check if their answer was correct

	for {
		fmt.Printf(
			"Pickan option:\n" + "" +
				"1. Show question.\n" +
				"2. Send answer.\n" +
				"3. Check if you answered correctly.\n" +
				"4. Exit application.\n",
		)

		//Read from STDIN
		switch utils.ReadStringStdin() {
		case "1":
			readQuestion(session)
			break
		case "2":
			fmt.Println("Type in your answer")
			sendAnswer(session, utils.ReadStringStdin())
			break
		case "3":
			checkCorrect(session)
			break
		case "4":
			fmt.Println("Exiting Application.... Goodbye")
			return
		default:
			fmt.Println("Invalid option. Please try again.")
			break
		}

	}
}

//Sessions are wrappers that allow us to make contract calls without having to pass aorund auth, creds,
//  and call parameters constantly
//A session wraps:
//  a contract instance
//	a bind.CallOpts struct that contains opts for making contract calls
//	a bind.TransactOpts struct that contains auth creds and params for created a valid Ethereum transaction

func NewSession(ctx context.Context) (session quiz.QuizSession) {
	// load environment vars
	loadEnv()

	// read data from keystore file
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

	//return session without contract instance - we'll do that after we've obtained a contract instance when we deploy a new contract
	//  or when we load an existing contract
	return quiz.QuizSession{
		TransactOpts: *auth,
		CallOpts:     callOpts,
	}
}

//******* DEPLOY AND LOAD CONTRACT ************
//NewContract deploys a contract if no existing contract exists and assigns to session
//A new contract takes the following params
//session: the client object, which we initialezed in main()
//question: a string containing the question we want the user to answer
//answer: the answer to the question
func NewContract(session quiz.QuizSession, client *ethclient.Client, question, answer string) quiz.QuizSession {
	log.Println("Creating new contract\n")
	loadEnv()

	// we do not want to send the question and answer as plain text in the contract
	// or else everone can see it. Use env variables instead

	// then encode the answer as a keccak256 hash before seding as posrt of the session.DeployQuiz() call.

	// hash answer before sending it over network
	ansHash := stringToKeccak256(answer)

	// deploy quizt to get the contract addr, transaction object, adnd instance
	contractAddress, tx, instance, err := quiz.DeployQuiz(&session.TransactOpts, client, question, ansHash)
	if err != nil {
		log.Fatalf("could not deploy contract: %v\n", err)
	}

	//print the adress of the transaction - this can be used to up on ethresan the progress of the transaction
	fmt.Println("Contract deployed! Wait for tx %s to be confirmed.\n", tx.Hash().Hex())

	session.Contract = instance

	//save the address of the deployed contract
	updateEnvFile("CONTRACTADDR", contractAddress.Hex())

	return session
}

//LoadContract creates or loads a contract if one exists, and assigns to the session
//attempts to load an existing contract by looking for a entry in the env file
func LoadContract(session quiz.QuizSession, client *ethclient.Client) quiz.QuizSession {
	log.Println("Loading contract\n")
	loadEnv()

	// check for existing contract
	addr := common.HexToAddress(os.Getenv("CONTRACTADDR"))

	// create new contract instance
	instance, err := quiz.NewQuiz(addr, client) //creates a new instance of Quiz, bound to a specific deployed contract
	if err != nil {
		// if addr doesn't exists we don't know where to locate our contract on the blockchain
		log.Fatalf("could not load contract: %v\n", err)
		//log.Println(ErrTransactionWait)
	}

	session.Contract = instance

	return session
}

func updateEnvFile(key, val string) {
	myenv[key] = val
	if err := godotenv.Write(myenv, envLoc); err != nil {
		log.Printf("failed to update %s: %s", envLoc, err)
	}
}

//stringToKeccak256 converts a string to a keccak256 hash of type [32]byte
func stringToKeccak256(s string) [32]byte {
	var toReturn [32]byte
	copy(toReturn[:], crypto.Keccak256([]byte(s))[:])
	return toReturn
}

// ********* Contract Interactions ************
//ErrTransactionWait should be returned when we encounter an error that may be a
// result of the transaction not being confirmed yet.
const ErrTransactionWait = "If you've just started the application, wait for the network to confirm your transaction."

//readQuestion prints out the question stored in the contract
func readQuestion(session quiz.QuizSession) {
	qn, err := session.Question()
	if err != nil {
		log.Println("could not read question from contract: ", err)
		log.Println(ErrTransactionWait)
		return
	}

	fmt.Println("Question: ", qn)
	return
}

//sendAnswer sends answer to the contract as a keccak256 hash
func sendAnswer(session quiz.QuizSession, ans string) {
	//send answer
	txSendAnswer, err := session.SendAnswer(stringToKeccak256(ans))
	if err != nil {
		log.Println("could not send answer to contract: ", err)
		return
	}

	fmt.Printf("Answer Sent! Please wait for tx %s to be confirmed.\n", txSendAnswer.Hash().Hex())
	return
}

//checkCorrect makes a contract message call to check if the current account owner has answered the question correctly
func checkCorrect(session quiz.QuizSession) {
	win, err := session.CheckBoard()
	if err != nil {
		log.Println("could not check leaderboard: ", err)
		log.Println(ErrTransactionWait)
		return
	}

	fmt.Println("Were you correct?: ", win)
	return
}
