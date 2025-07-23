package main

import (
	"fmt"
	"log"
)

func main() {
	phone := fmt.Sprintf("%010d", 1100) // 10 digits: 0000001000 -> 0000009999
	log.Println(phone)
}