package main

import (
	"fmt"
)

func main() {
	b := make([]byte, 4)
	fmt.Println(cap(b))
}