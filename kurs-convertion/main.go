package main

import "fmt"

type ConversionResult struct {
	Currency string
	Amount   float64
}

func main() {
	var amount float64
	fmt.Print("Masukkan jumlah uang dalam rupiah: ")
	fmt.Scan(&amount)
	
	kurs := map[string]float64{
		"USD": 15000.00,
		"EUR": 16000.00,
		"JPY": 140.00,
		"SGD": 11000.00,
	}
	result := []ConversionResult{}

	for currency, rate := range kurs {
		convertedAmount := ConversionResult{
			Currency: currency,
			Amount:   amount / rate,
		}
		result = append(result, convertedAmount)
	}

	fmt.Println("Konversi dari", amount, "IDR:")
	for _, res := range result {
		fmt.Printf("%s: %.3f\n", res.Currency, res.Amount)
	}
}