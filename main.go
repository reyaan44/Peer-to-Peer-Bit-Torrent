package main

import (
	"fmt"
	"os"

	gotorrentparser "github.com/j-muller/go-torrent-parser"
)

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

}
