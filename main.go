package main

import (
	"fmt"
	"os"
	"sync"

	gotorrentparser "github.com/j-muller/go-torrent-parser"
	bencode "github.com/jackpal/bencode-go"
)

var wg = sync.WaitGroup{}

var myPeerId []byte

var pieceCount int
var pieceSize int

func main() {

	myPeerId = generateRandomBytes(20)

	arg := os.Args[:]
	filePath := fmt.Sprintf("./torrents/%s", arg[1])

	// Getting the piece size and the number of pieces
	file, err := os.Open(filePath)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	bencode_info := bencodeTorrentFiles{}
	err = bencode.Unmarshal(file, &bencode_info)
	if err != nil {
		panic(err)
	}
	/*
		Info.Pieces = A string of all the SHA1 hash values of the pieces. The length of this string is therefore a multiple of 20.
		Info.PieceLength = The length of each piece, in bytes.
		Info.Length = The length of the file, in bytes.
		Last Piece Length = Info.Length % Info.PieceLength (if the last piece is not the same size as the other pieces)
	*/
	pieceSize = bencode_info.Info.PieceLength
	pieceCount = len(bencode_info.Info.Pieces) / 20
	totalLength := bencode_info.Info.Length

	fileList := []File{}

	if totalLength == 0 {
		fmt.Println("Multifile Torrent")
		for idx, file := range bencode_info.Info.Files {
			filePath := "./Reyaan-Downloads/"
			for _, path := range file.Path {
				if path != file.Path[len(file.Path)-1] {
					filePath += path + "/"
				} else {
					err := os.MkdirAll(filePath, 0750)
					if err != nil && !os.IsExist(err) {
						fmt.Println(err)
						return
					}
					filePath += path
				}
			}
			fileList = append(fileList, File{index: idx, path: filePath, length: file.Length})
			totalLength += file.Length
		}
	} else {
		fmt.Println("Single File Torrent")
		filePath := "./Reyaan-Downloads/"
		err := os.MkdirAll(filePath, 0750)
		if err != nil && !os.IsExist(err) {
			fmt.Println(err)
			return
		}
		filePath += bencode_info.Info.Name
		fileList = append(fileList, File{index: 0, path: filePath, length: totalLength})
	}
	lastPieceSize := totalLength % bencode_info.Info.PieceLength
	piecesSha1Hashed := bencode_info.Info.Pieces
	fmt.Println("Piece Size : ", pieceSize)
	fmt.Println("Piece Count : ", pieceCount)
	fmt.Println("Total Length : ", totalLength)
	fmt.Println("Last Piece Size : ", lastPieceSize)

	// Parsing the torrent using gotorrentparser
	Torrent, err := gotorrentparser.ParseFromFile(filePath)
	if err != nil {
		panic(err)
	}

	peersList := getPeersList(Torrent)
	fmt.Println(peersList)
	fmt.Println("Total Number of Peers : ", len(peersList))

	// For every peer in the list, start a new goroutine for tcp connection
	peerConnectionTcpList := []PeerConnection{}
	for current := range peersList {
		wg.Add(1)
		go SendHandshake(peersList[current], Torrent, &peerConnectionTcpList)
	}
	wg.Wait()
	fmt.Println(peerConnectionTcpList)
	fmt.Println("Total Number of Successful Peers : ", len(peerConnectionTcpList))

	pieces := make([]*Piece, pieceCount)

	for i := 0; i < pieceCount; i++ {
		pieces[i] = &Piece{}
		pieces[i].index = uint32(i)
		copy(pieces[i].hash[:], []byte(piecesSha1Hashed[i*20:(i+1)*20]))
		if i+1 == pieceCount && lastPieceSize > 0 {
			pieces[i].length = uint32(lastPieceSize)
		} else {
			pieces[i].length = uint32(pieceSize)
		}
		pieces[i].data = nil
	}

	// For each piece, write the corresponding files along with their offsets for the files
	setFilePieceOffset(pieces, fileList)

	// getting the bitfield of each peer
	for i := range peerConnectionTcpList {
		peerConnectionTcpList[i].bitfield = make([]bool, pieceCount)
	}

	QueueNeededPieces := make(chan *Piece, pieceCount)
	QueueFinishedPieces := make(chan *Piece, pieceCount)
	defer close(QueueNeededPieces)
	defer close(QueueFinishedPieces)

	for i := range pieces {
		QueueNeededPieces <- pieces[i]
	}

	for i := range peerConnectionTcpList {
		wg.Add(1)
		go startNewDownload(&peerConnectionTcpList[i], Torrent, QueueNeededPieces, QueueFinishedPieces, pieces)
	}
	wg.Wait()

	if len(QueueFinishedPieces) == pieceCount {
		fmt.Println("Download Finished and File Created")
	}

	for i := range peerConnectionTcpList {
		peerConnectionTcpList[i].connId.Close()
	}

}
