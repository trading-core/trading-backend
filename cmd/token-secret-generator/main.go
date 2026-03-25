package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

func main() {
	key := make([]byte, 32) // 32 bytes = 256 bits
	_, err := rand.Read(key)
	if err != nil {
		panic(err)
	}
	secret := base64.StdEncoding.EncodeToString(key)
	fmt.Println("Token Secret:", secret)
}
