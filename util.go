package main

import (
	"math/rand"
	"time"
)

func generateRandomBytes(n int) []byte {
	// Set up random seed using current timestamp and a random number
	seed := time.Now().UnixNano() + int64(rand.Intn(1000000))
	rand.Seed(seed)

	// Generate random byte slice of length n
	randBytes := make([]byte, n)
	rand.Read(randBytes)

	return randBytes
}

func min(a int, b int) int {
	if a < b {
		return a
	}
	return b
}
