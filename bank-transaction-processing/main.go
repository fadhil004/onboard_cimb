package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"
)

type Account struct {
	Bank  string
	No    string
	Saldo int
}

type Transaction struct {
	ID     int
	From   string
	To     string
	Amount int
}

var (
	accounts = map[string]*Account{		
		"C001": {"CIMB", "C001", 300000},
		"M002": {"MANDIRI", "M002", 500000},
		"B003": {"BNI", "B003", 400000},
		"BC04": {"BCA", "BC04", 800000},
	}
	mu sync.Mutex
)

// Worker
func processTransaction(trx Transaction, wg *sync.WaitGroup) {
	defer wg.Done()

	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()

	done := make(chan bool)

	go func() {
		delay := time.Duration(rand.Intn(5)+1) * time.Second

		time.Sleep(delay)

		mu.Lock()
		defer mu.Unlock()

		from := accounts[trx.From]
		to := accounts[trx.To]

		log.Printf("| Processing TX-%d | %s -> %s | Rp%d", trx.ID, from.No, to.No, trx.Amount)

		if from.Saldo < trx.Amount {
			log.Printf("| TX-%d FAILED | %s -> %s | Transfer Rp%d | Saldo %s hanya Rp%d",
				trx.ID, from.No, to.No, trx.Amount, from.No, from.Saldo)
		} else {
			from.Saldo -= trx.Amount
			to.Saldo += trx.Amount

			log.Printf("| TX-%d SUCCESS \n Saldo %s sekarang Rp%d \n Saldo %s sekarang Rp%d",
				trx.ID, from.No, from.Saldo, to.No, to.Saldo)
		}

		done <- true
	}()

	select {
	case <-ctx.Done():
		log.Printf("TX-%d TIMEOUT", trx.ID)
	case <-done:
		// selesai normal
	}
}

func main() {
	rand.Seed(time.Now().UnixNano())

	transactions := []Transaction{
		{1, "C001", "M002", 300000},
		{2, "C001", "B003", 600000},
		{3, "M002", "BC04", 200000},
		{4, "B003", "C001", 500000},
		{5, "BC04", "M002", 700000},
		{6, "C001", "M002", 400000},
		{7, "B003", "C001", 300000},
	}

	var wg sync.WaitGroup

	for _, trx := range transactions {
		wg.Add(1)
		go processTransaction(trx, &wg)
	}

	wg.Wait()

	fmt.Println("\n=== SALDO AKHIR ===")
	for _, acc := range accounts {
		fmt.Printf("%s (%s): Rp%d \n", acc.Bank, acc.No, acc.Saldo)
	}
}