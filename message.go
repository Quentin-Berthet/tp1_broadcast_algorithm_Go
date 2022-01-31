package main

import (
	"encoding/json"
	"fmt"
	"io"
)

type MessageType string

const (
	CreateTransaction MessageType = "CreateTransaction"
	RateTransaction   MessageType = "RateTransaction"
	Acknowledge       MessageType = "Acknowledge"
	ListTransactions  MessageType = "ListTransactions"
	FakeTransaction   MessageType = "FakeTransaction"
)

type Message struct {
	Type        MessageType
	Transaction Transaction
	Rates       map[string]bool
	Transactions []Transaction
}

func NewMessage(mType MessageType, transaction Transaction, rates map[string]bool, transactions []Transaction) Message {
	return Message{Type: mType, Transaction: transaction, Rates: rates, Transactions: transactions}
}

func MessageToJSON(message Message) ([]byte, error) {
	return json.Marshal(message)
}

func MessageFromJSON(r io.Reader) (Message, error) {
	decoder := json.NewDecoder(r)
	var message Message
	var err = decoder.Decode(&message)
	return message, err
}

func MessageToString(message Message) string {
	return fmt.Sprintf("Message(%+v)", message)
}
