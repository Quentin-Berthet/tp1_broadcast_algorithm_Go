package main

/*
	Author: Baptiste Coudray
	Date: 14.10.2020
	Course: Distributed Systems
	Description: Go version of broadcast algorithm
*/

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"strings"
	"sync"
)

func parseNeighboursFile(filename string) []string {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	trimmedData := strings.Trim(string(data), "\n")
	return strings.Split(trimmedData, "\n")
}

func parseRootingFile(filename string) map[string]string {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	routingTable := make(map[string]string)
	trimmedData := strings.Trim(string(data), "\n")
	lines := strings.Split(trimmedData, "\n")
	for _, line := range lines {
		information := strings.Split(line, " ")
		routingTable[information[0]] = information[1]
	}
	return routingTable
}

var database = NewDatabase()
var crNeighboursReached = make(map[Transaction][]string)
var crNeighboursReachedMutex sync.Mutex
var databaseMutex sync.Mutex

func arrayStrContains(needle string, haystack []string) bool {
	for _, a := range haystack {
		if a == needle {
			return true
		}
	}
	return false
}

func sendTransactionMessageTo(neighbour string, message Message) {
	ipPort := strings.Split(neighbour, ":")
	if len(ipPort) == 1 {
		ipPort = append(ipPort, connPort)
	}
	client, errDial := net.Dial(connType, fmt.Sprintf("%s:%s", ipPort[0], ipPort[1]))
	if errDial == nil {
		data, _ := MessageToJSON(message)
		_, errWrite := client.Write(data)
		if errWrite == nil {
			fmt.Printf("[sendMessageTo] I sent this %s to my neighbour %s.\n", TransactionToString(message.Transaction), neighbour)
		} else {
			fmt.Printf("[sendMessageTo] Unable to write data to %s. (%s)\n", neighbour, errWrite.Error())
		}
	} else {
		fmt.Printf("[sendMessageTo] Unable to dial %s. (%s)\n", neighbour, errDial.Error())
	}
}

func createTransaction(transaction Transaction, neighbours []string) {
	databaseMutex.Lock()
	database = DatabaseAdd(database, transaction)
	fmt.Printf("[createTransaction] User has created this %s.\n", TransactionToString(transaction))
	databaseMutex.Unlock()
	for _, neighbour := range neighbours {
		crNeighboursReachedMutex.Lock()
		crNeighboursReached[transaction] = append(crNeighboursReached[transaction], neighbour)
		crNeighboursReachedMutex.Unlock()
		go sendTransactionMessageTo(neighbour, NewMessage(CreateTransaction, transaction, nil, nil))
	}
}

func createTransactionOnReceived(message Message) {
	databaseMutex.Lock()
	if !DatabaseContains(database, message.Transaction) {
		database = DatabaseAdd(database, message.Transaction)
		fmt.Printf("[createTransactionOnReceived] I added this %s to my DB.\n", TransactionToString(message.Transaction))
	}
	databaseMutex.Unlock()
	for _, neighbour := range neighbours {
		crNeighboursReachedMutex.Lock()
		neighboursAlreadyReached := crNeighboursReached[message.Transaction]
		if !arrayStrContains(neighbour, neighboursAlreadyReached) {
			crNeighboursReached[message.Transaction] = append(crNeighboursReached[message.Transaction], neighbour)
			fmt.Printf("[createTransactionOnReceived] I reached this neighbour %s.\n", neighbour)
			fmt.Printf("[createTransactionOnReceived] So far, I reached this neighbours %s.\n", crNeighboursReached[message.Transaction])
			crNeighboursReachedMutex.Unlock()
			go sendTransactionMessageTo(neighbour, message)
		} else {
			crNeighboursReachedMutex.Unlock()
		}
	}
}

var transactionToRate Transaction
var ackMessages = make([]Message, 0)
var expectedAck = 0
var parents = make([]string, 0)
var rateMutex sync.Mutex

func rateTransactionOnReceived(connParent net.Conn, message Message) {
	parentAddr := strings.Split(connParent.RemoteAddr().String(), ":")
	rateMutex.Lock()
	parents = append(parents, parentAddr[0])
	fmt.Printf("[rateTransactionOnReceived] I added this parent %s.\n", parentAddr[0])
	fmt.Printf("[rateTransactionOnReceived] So far, I have this parents %v.\n", parents)
	hasChildren := false
	rateMutex.Unlock()
	for _, neighbour := range neighbours {
		rateMutex.Lock()
		if !arrayStrContains(neighbour, parents) {
			connChild, err := net.Dial(connType, fmt.Sprintf("%s:%s", neighbour, connPort))
			if err == nil {
				data, _ := MessageToJSON(message)
				_, err := connChild.Write(data)
				if err == nil {
					expectedAck += 1
					hasChildren = true
					fmt.Printf("[rateTransactionOnReceived] I sent this %s from %s.\n", MessageToString(message), connChild.RemoteAddr())
					go waitAcknowledgeMessage(connParent, connChild)
				} else {
					fmt.Printf("[rateTransactionOnReceived] Unable to write data to %s. (%s)\n", neighbour, err.Error())
				}
			} else {
				fmt.Printf("[rateTransactionOnReceived] Unable to dial %s. (%s)\n", neighbour, err.Error())
			}
		}
		rateMutex.Unlock()
	}
	if !hasChildren {
		localAddr := strings.Split(connParent.LocalAddr().String(), ":")
		/* I don't have any children so this is my job to create an ACK message */
		rates := make(map[string]bool)
		databaseMutex.Lock()
		rates[localAddr[0]] = DatabaseContains(database, message.Transaction)
		databaseMutex.Unlock()
		ackMessage := NewMessage(Acknowledge, message.Transaction, rates, nil)
		data, _ := MessageToJSON(ackMessage)
		_, err := connParent.Write(data)
		if err == nil {
			fmt.Printf("[rateTransactionOnReceived] No children, I sent this %s to %s.\n", MessageToString(ackMessage), parentAddr[0])
		} else {
			fmt.Printf("[rateTransactionOnReceived] Unable to write data to %s. (%s)\n", parentAddr[0], err.Error())
		}
		/* I sent an ACK I can reset my internal values to accept an other rate demand */
		rateMutex.Lock()
		parents = make([]string, 0)
		ackMessages = make([]Message, 0)
		expectedAck = 0
		rateMutex.Unlock()
		_ = connParent.Close()
	}
}

func rateTransaction(transaction Transaction, neighbours []string) {
	rateMutex.Lock()
	transactionToRate = transaction
	expectedAck = len(neighbours)
	rateMutex.Unlock()
	for _, neighbour := range neighbours {
		conn, errDial := net.Dial(connType, fmt.Sprintf("%s:%s", neighbour, connPort))
		if errDial == nil {
			message := NewMessage(RateTransaction, transaction, nil, nil)
			data, _ := MessageToJSON(message)
			_, errWrite := conn.Write(data)
			if errWrite == nil {
				fmt.Printf("[rateTransaction] I sent this %s to my neighbour %s.\n", MessageToString(message), neighbour)
				go waitAcknowledgeMessage(nil, conn)
			} else {
				fmt.Printf("[rateTransaction] Unable to write data to %s. (%s)\n", neighbour, errWrite.Error())
			}
		} else {
			fmt.Printf("[rateTransaction] Unable to dial %s. (%s)\n", neighbour, errDial.Error())
		}
	}
}

func mergeRatesMap(left map[string]bool, right map[string]bool) {
	for key, value := range right {
		if _, present := left[key]; !present {
			left[key] = value
		}
	}
}

func computeRate(rates map[string]bool) float64 {
	nbTrue := 0
	for _, valid := range rates {
		if valid {
			nbTrue += 1
		}
	}
	return float64(nbTrue) / float64(len(rates)) * 100.0
}

func waitAcknowledgeMessage(connParent net.Conn, connChild net.Conn) {
	fmt.Printf("[waitAcknowledgeMessage] I'm waiting an ack from %s.\n", connChild.RemoteAddr())
	message, err := MessageFromJSON(connChild)
	fmt.Printf("[waitAcknowledgeMessage] I received this %s from %s.\n", MessageToString(message), connChild.RemoteAddr())
	if err == nil {
		localAddr := strings.Split(connChild.LocalAddr().String(), ":")
		databaseMutex.Lock()
		message.Rates[localAddr[0]] = DatabaseContains(database, message.Transaction)
		databaseMutex.Unlock()
		rateMutex.Lock()
		ackMessages = append(ackMessages, message)
		if len(ackMessages) == expectedAck {
			/* I receive all ACKs, I can merge all rates map into one map */
			for i := 1; i < len(ackMessages); i++ {
				mergeRatesMap(ackMessages[0].Rates, ackMessages[i].Rates)
			}
			if len(parents) == 0 {
				fmt.Printf("[waitAcknowledgeMessage] I have received all acks. The rates for %s are %v.\n", TransactionToString(ackMessages[0].Transaction), ackMessages[0].Rates)
				rate := computeRate(ackMessages[0].Rates)
				fmt.Printf("[waitAcknowledgeMessage] This %s is valid at %.2f %%.\n", TransactionToString(transactionToRate), rate)
			} else {
				data, _ := MessageToJSON(ackMessages[0])
				_, err := connParent.Write(data)
				if err == nil {
					fmt.Printf("[waitAcknowledgeMessage] I sent this %s to my parent %s.\n", MessageToString(message), connParent.RemoteAddr())
				} else {
					fmt.Printf("[waitAcknowledgeMessage] Unable to write data to %s. (%s)\n", connParent.RemoteAddr(), err.Error())
				}
				_ = connParent.Close()
			}
			/* Reset internal values to be able to handle a new rate demand */
			parents = make([]string, 0)
			ackMessages = make([]Message, 0)
			expectedAck = 0
		} else {
			fmt.Printf("[waitAcknowledgeMessage] I'm waiting for %d ack, I have received %d ack.\n", expectedAck, len(ackMessages))
		}
		rateMutex.Unlock()
	} else {
		fmt.Printf("[waitAcknowledgeMessage] Failed to read ACK message from %s.\n", connChild.RemoteAddr().String())
	}
	_ = connChild.Close()
}

func listOfTransactionsOnReceived(conn net.Conn, message Message) {
	databaseMutex.Lock()
	transactions := make([]Transaction, 0)
	for _, transaction := range database.Transactions {
		transactions = append(transactions, transaction)
	}
	databaseMutex.Unlock()
	message.Transactions = transactions
	data, _ := MessageToJSON(message)
	_, err := conn.Write(data)
	if err == nil {
		fmt.Printf("[listOfTransactionsOnReceived] I sent the list of transactions to %s.\n", conn.RemoteAddr())
	} else {
		fmt.Printf("[listOfTransactionsOnReceived] Unable to write data to %s. (%s)\n", conn.RemoteAddr(), err.Error())
	}
	_ = conn.Close()
}

func fakeTransactionOnReceived(conn net.Conn, message Message) {
	databaseMutex.Lock()
	database.Transactions[message.Transaction.Id] = message.Transaction
	databaseMutex.Unlock()
	_ = conn.Close()
}

var (
	connHost = "0.0.0.0"
	connPort = "1337"
	connType = "tcp"
	nodeId   = "0"
)

var neighbours []string

func main() {
	if len(os.Args) < 3 {
		fmt.Printf("Usage: %s nodeId port", os.Args[0])
		os.Exit(1)
	}
	nodeId = os.Args[1]
	connPort = os.Args[2]

	neighbours = parseNeighboursFile("neighbour-" + nodeId + ".txt")
	fmt.Println("Neighbours = ", neighbours)
	go launchServer()
	for {
		handleMenu()
	}
}

// https://dev.to/alicewilliamstech/getting-started-with-sockets-in-golang-2j66
func launchServer() {
	listener, err := net.Listen(connType, fmt.Sprintf("%s:%s", connHost, connPort))
	if err != nil {
		fmt.Println("[launchServer] Error listening:", err.Error())
		listener.Close()
		os.Exit(1)
	}
	defer listener.Close()
	fmt.Printf("[launchServer] Listening on %s://%s:%s...\n", connType, connHost, connPort)
	for {
		client, err := listener.Accept()
		if err == nil {
			fmt.Printf("[launchServer] Client %s connected.\n", client.RemoteAddr())
			go handleClient(client)
		} else {
			fmt.Printf("[launchServer] Error connecting: %s.\n", err.Error())
		}
	}
}

func handleClient(conn net.Conn) {
	message, err := MessageFromJSON(conn)
	if err == nil {
		fmt.Printf("[handleConnection] I decoded this %s from %s.\n", MessageToString(message), conn.RemoteAddr())
		switch message.Type {
		case CreateTransaction:
			createTransactionOnReceived(message)
			_ = conn.Close()
		case RateTransaction:
			rateTransactionOnReceived(conn, message)
		case ListTransactions:
			listOfTransactionsOnReceived(conn, message)
		case FakeTransaction:
			fakeTransactionOnReceived(conn, message)
		}
	} else {
		fmt.Printf("[handleConnection] I couldn't parse JSON to Message. (%s).\n", err.Error())
		_ = conn.Close()
	}
}
