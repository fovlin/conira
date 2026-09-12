package main

import (
	"encoding/binary"
	"fmt"
)

func main() {
	b := make([]byte, 4)
	var v uint32 = 32
	binary.LittleEndian.PutUint32(b,v)
	fmt.Println(b)
}