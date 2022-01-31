package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func stdioCreateTransaction() {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Sender: ")
	sender, _ := reader.ReadString('\n')
	sender = strings.Trim(sender, "\n")
	fmt.Print("Receiver: ")
	receiver, _ := reader.ReadString('\n')
	receiver = strings.Trim(receiver, "\n")
	fmt.Print("Amount: ")
	amount, _ := reader.ReadString('\n')
	amount = strings.Trim(amount, "\n")
	floatAmount, _ := strconv.ParseFloat(amount, 64)
	createTransaction(NewTransaction(sender, receiver, floatAmount), neighbours)
}

func stdioRateTransaction() {
	databaseMutex.Lock()
	fmt.Printf("%s", DatabaseToString(database))
	databaseMutex.Unlock()
	fmt.Println("Which transaction would you like to rate?")
	fmt.Print("Id of the transaction: ")
	reader := bufio.NewReader(os.Stdin)
	id, _ := reader.ReadString('\n')
	id = strings.Trim(id, "\n")
	databaseMutex.Lock()
	transaction, exist := database.Transactions[id]
	databaseMutex.Unlock()
	if exist {
		rateTransaction(transaction, neighbours)
	}
}

func handleMenu() {
	fmt.Println("What do you want to do?")
	fmt.Println("(1) Create a transaction")
	fmt.Println("(2) Rate a transaction")
	reader := bufio.NewReader(os.Stdin)
	choice, _ := reader.ReadString('\n')
	choice = strings.Trim(choice, "\n")
	switch choice {
	case "1":
		stdioCreateTransaction()
	case "2":
		stdioRateTransaction()
	}
}
