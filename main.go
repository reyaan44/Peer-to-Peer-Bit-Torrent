package main

import (
	"fmt"
	"os"
	"sync"

	gotorrentparser "github.com/j-muller/go-torrent-parser"
)

var wg = sync.WaitGroup{}

var myPeerId []byte

func main() {

	myPeerId = generateRandomBytes(20)

	arg := os.Args[:]
	filePath := fmt.Sprintf("./torrents/%s", arg[1])

	// Pasring the torrent using gotorrentparser
	Torrent, err := gotorrentparser.ParseFromFile(filePath)
	if err != nil {
		panic(err)
	}

	peersList := getPeersList(Torrent)
	fmt.Println(peersList)
	fmt.Println("TOTAL NUMBER OF PEERS : ", len(peersList))

	// For every peer in the list, start a new goroutine for tcp connection
	peerConnectionTcpList := []PeerConnection{}
	for current := range peersList {
		wg.Add(1)
		go SendHandshake(peersList[current], Torrent, &peerConnectionTcpList)
	}
	wg.Wait()
	fmt.Println(peerConnectionTcpList)
	fmt.Println("TOTAL NUMBER OF SUCCESSFUL PEERS : ", len(peerConnectionTcpList))

	for i := range peerConnectionTcpList {
		wg.Add(1)
		go startNewDownload(peerConnectionTcpList[i], Torrent)
	}
	wg.Wait()
}
