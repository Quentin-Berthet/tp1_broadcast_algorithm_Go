package main

import "fmt"

type Database struct {
	Transactions map[string]Transaction
}

func NewDatabase() Database {
	return Database{Transactions: make(map[string]Transaction, 0)}
}

func DatabaseContains(database Database, needle Transaction) bool {
	transaction, exists := database.Transactions[needle.Id]
	return exists && transaction == needle
}

func DatabaseAdd(database Database, transaction Transaction) Database {
	database.Transactions[transaction.Id] = transaction
	return database
}

func DatabaseToString(database Database) string {
	output := ""
	for _, transaction := range database.Transactions {
		output += fmt.Sprintf("%s\n", TransactionToString(transaction))
	}
	return output
}
