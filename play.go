package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
)

const network = "tcp"

var ip = ""
var port = ""

func fakeTransaction() {
	fmt.Println("Which transaction to fake ?")
	transactions := listOfTransactions()
	showTransactions(transactions, true)

	fmt.Print("Index of the transaction: ")
	reader := bufio.NewReader(os.Stdin)
	id, _ := reader.ReadString('\n')
	id = strings.Trim(id, "\n")
	index, _ := strconv.Atoi(id)
	index -= 1

	if index >= 0 && index < len(transactions) {
		transaction := transactions[index]

		fmt.Print("Sender: ")
		sender, _ := reader.ReadString('\n')
		transaction.Sender = strings.Trim(sender, "\n")
		fmt.Print("Receiver: ")
		receiver, _ := reader.ReadString('\n')
		transaction.Receiver = strings.Trim(receiver, "\n")
		fmt.Print("Amount: ")
		amount, _ := reader.ReadString('\n')
		amount = strings.Trim(amount, "\n")
		floatAmount, _ := strconv.ParseFloat(amount, 64)
		transaction.Amount = floatAmount

		message := NewMessage(FakeTransaction, transaction, nil, nil)
		conn, err := net.Dial(network, fmt.Sprintf("%s:%s", ip, port))
		if err != nil {
			fmt.Printf("[fakeTransaction] Unable to dial %s. (%s)\n", ip, err.Error())
			return
		}
		data, _ := MessageToJSON(message)
		_, err = conn.Write(data)
		if err == nil {
			fmt.Println("[fakeTransaction] Transaction has been faked.")
		} else {
			fmt.Printf("[fakeTransaction] Unable to write data to %s. (%s)\n", ip, err.Error())
			return
		}
		conn.Close()
	}
}

func listOfTransactions() []Transaction {
	transactions := make([]Transaction, 0)
	conn, err := net.Dial(network, fmt.Sprintf("%s:%s", ip, port))
	if err != nil {
		fmt.Printf("[listOfTransactions] Unable to dial %s. (%s)\n", ip, err.Error())
		return transactions
	}
	defer conn.Close()
	message := NewMessage(ListTransactions, Transaction{}, nil, nil)
	data, _ := MessageToJSON(message)
	_, err = conn.Write(data)
	if err != nil {
		fmt.Printf("[listOfTransactions] Unable to write data to %s. (%s)\n", ip, err.Error())
		return transactions
	}
	message, err = MessageFromJSON(conn)
	if err != nil {
		fmt.Printf("[listOfTransactions] I couldn't parse JSON to Message. (%s).\n", err.Error())
		return transactions
	}
	return message.Transactions
}

func showTransactions(transactions []Transaction, withIndex bool) {
	for i, transaction := range transactions {
		if withIndex {
			fmt.Printf("(%d) %s\n", i+1, TransactionToString(transaction))
		} else {
			fmt.Println(TransactionToString(transaction))
		}
	}
}

func main() {
	if len(os.Args) < 3 {
		fmt.Println("You need to specify an IP and a port of a server.")
		os.Exit(1)
	}
	ip = os.Args[1]
	port = os.Args[2]
	for {
		fmt.Println("What do you want to do ?")
		fmt.Println("(1) Fake a transaction")
		fmt.Println("(2) List of transactions")
		reader := bufio.NewReader(os.Stdin)
		choice, _ := reader.ReadString('\n')
		choice = strings.Trim(choice, "\n")
		switch choice {
		case "1":
			fakeTransaction()
		case "2":
			transactions := listOfTransactions()
			showTransactions(transactions, false)
		}
	}
}
