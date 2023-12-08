package main

import (
	"fmt"
	"log"
	zcrypto "terylene/crypto"
)

func main() {
	key, err := zcrypto.GenerateRandomKey(32)
	if err != nil {
		log.Fatalln(err)
	}

	fmt.Println(string(key))
}
