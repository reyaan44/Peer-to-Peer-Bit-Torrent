package main

import (
	"fmt"
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

func max(a int, b int) int {
	if a < b {
		return a
	}
	return b
}

func waitTimeReConnection(successfulPeers int) int {
	waitTimeReConnection := 300
	// switch {
	// case successfulPeers < 0:
	// 	fmt.Println("Negative")
	// 	break
	// case successfulPeers <= 10:
	// 	waitTimeReConnection = 60
	// 	break
	// case successfulPeers <= 20:
	// 	waitTimeReConnection = 90
	// 	break
	// case successfulPeers <= 30:
	// 	waitTimeReConnection = 110
	// 	break
	// case successfulPeers <= 50:
	// 	waitTimeReConnection = 120
	// 	break
	// case successfulPeers <= 100:
	// 	waitTimeReConnection = 150
	// 	break
	// default:
	// 	waitTimeReConnection = 180
	// 	break
	// }
	return waitTimeReConnection
}

func waitTimeBitField(successfulPeers int) int {
	waitTimeBitField := 60
	switch {
	case successfulPeers < 0:
		fmt.Println("Negative")
		break
	case successfulPeers <= 10:
		waitTimeBitField = 30
		break
	case successfulPeers <= 20:
		waitTimeBitField = 60
		break
	case successfulPeers <= 30:
		waitTimeBitField = 90
		break
	case successfulPeers <= 50:
		waitTimeBitField = 120
		break
	case successfulPeers <= 100:
		waitTimeBitField = 240
		break
	default:
		waitTimeBitField = 300
		break
	}
	return waitTimeBitField
}

func waitTimeReceive(successfulPeers int) int {
	waitTimeReceive := 10
	switch {
	case successfulPeers < 0:
		fmt.Println("Negative")
		break
	case successfulPeers <= 10:
		waitTimeReceive = 10
		break
	case successfulPeers <= 20:
		waitTimeReceive = 15
		break
	case successfulPeers <= 30:
		waitTimeReceive = 20
		break
	case successfulPeers <= 50:
		waitTimeReceive = 20
		break
	case successfulPeers <= 100:
		waitTimeReceive = 30
		break
	default:
		waitTimeReceive = 60
		break
	}
	return waitTimeReceive
}
