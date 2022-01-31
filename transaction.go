package main

import (
	"fmt"
	guuid "github.com/google/uuid"
)

type Transaction struct {
	Id       string
	Sender   string
	Receiver string
	Amount   float64
}

func NewTransaction(sender string, receiver string, amount float64) Transaction {
	return Transaction{Id: guuid.New().String(), Sender: sender, Receiver: receiver, Amount: amount}
}

func TransactionToString(transaction Transaction) string {
	return fmt.Sprintf("Transaction(%+v)", transaction)
}
