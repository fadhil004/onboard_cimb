package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

type Card struct {
	Type   string 
	Prefix []string
	Length []int
}


func main() {
	scanner := bufio.NewReader(os.Stdin)
	var input []string
	fmt.Println("Masukkan nomor kartu: ")
	for {
		line, _:= scanner.ReadString('\n')
		line = strings.TrimSpace(line)
		if line == "" {
			break
		}
		input = append(input, line)
	}
	
	card := []Card{{
		Type: "China Unionpay",
		Prefix: []string{"62"},
		Length: []int{16,17,18,19},
		},
		{
		Type: "Switch",
		Prefix: []string{"4903", "4905", "4911", "4936", "564182", "633110", "6333", "6759"},
		Length: []int{16,18,19},
		},
	}

	for _, i := range input {
		found := false
		for _, c := range card {
			for _, p := range c.Prefix {
				if strings.HasPrefix(i, p) {
					for _, l := range c.Length {
						if len(i) == l {
							fmt.Println("Nomor", i, "adalah",c.Type)
							found = true
							break;
						}
					}
				} 
			}
			if(found) {
				break;
			}
		}
		
		if(!found) {
			fmt.Println("Jenis kartu tidak dikenali")
		}
	}	
}