package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println("hello world")

	key := os.Getenv("BYBIT_API_KEY")
	fmt.Println("key is ", key)
}
