package main

import (
	"fmt"
	"os"
	"sync"
	"time"

	gotorrentparser "github.com/j-muller/go-torrent-parser"
	bencode "github.com/jackpal/bencode-go"
)

var wg = sync.WaitGroup{}
var wgRebuild = sync.WaitGroup{}

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
		go SendHandshake(&peersList[current], Torrent, &peerConnectionTcpList, false)
	}
	wg.Wait()
	successfulPeers := len(peerConnectionTcpList)
	fmt.Println("Total Number of Successful Peers : ", successfulPeers)

	QueueNeededPieces := make(chan *Piece, pieceCount)
	QueueFinishedPieces := make(chan *Piece, pieceCount)
	defer close(QueueNeededPieces)
	defer close(QueueFinishedPieces)

	for i := range pieces {
		QueueNeededPieces <- pieces[i]
	}

	for i := range peerConnectionTcpList {
		peerConnectionTcpList[i].peer.InsideQueue = true
		wg.Add(1)
		go startNewDownload(&peerConnectionTcpList[i], Torrent, QueueNeededPieces, QueueFinishedPieces, pieces)
	}

	done := make(chan bool) // Create a done channel

	go func() {
		wg.Wait() // Wait for all downloads to complete
		if len(QueueFinishedPieces) == pieceCount {
			fmt.Println("Download Finished")
		} else {
			fmt.Println("Download Not Finished")
		}
		done <- true // Send a message on the done channel when all downloads have finished
	}()

	waitTime := 180
	switch {
	case successfulPeers < 0:
		fmt.Println("Negative")
		break
	case successfulPeers <= 10:
		waitTime = 60
		break
	case successfulPeers <= 20:
		waitTime = 70
		break
	case successfulPeers <= 30:
		waitTime = 80
		break
	case successfulPeers <= 50:
		waitTime = 90
		break
	case successfulPeers <= 100:
		waitTime = 150
		break
	default:
		waitTime = 180
		break
	}

downloadLoop:
	for {
		select {
		case <-time.After(time.Duration(waitTime) * time.Second):

			fmt.Println("ReHandshaking and Reconnecting")

			reBuildGetPeersList(Torrent, &peersList)

			fmt.Println("Total Number of Peers Currently : ", len(peersList))

			// For every peer in the list, start a new goroutine for tcp connection
			for current := range peersList {
				if peersList[current].Handshake == true {
					continue
				}
				wgRebuild.Add(1)
				go SendHandshake(&peersList[current], Torrent, &peerConnectionTcpList, true)
			}
			wgRebuild.Wait()

			fmt.Println("New Total Number of Successful Peers Currently Added : ", len(peerConnectionTcpList)-successfulPeers)
			successfulPeers = len(peerConnectionTcpList)

			for current := range peerConnectionTcpList {
				if peerConnectionTcpList[current].peer.Handshake == true && peerConnectionTcpList[current].peer.InsideQueue == false {
					peerConnectionTcpList[current].peer.InsideQueue = true
					wg.Add(1)
					go startNewDownload(&peerConnectionTcpList[current], Torrent, QueueNeededPieces, QueueFinishedPieces, pieces)
				}
			}
			break

		case <-done:
			// All downloads have finished
			break downloadLoop
		}
	}
	// Interesting Case, If all the pieces are downloaded, but then just before that, rebuild is called, it will still send a startDownload message
	wg.Wait()

	// Close all the connections
	fmt.Println("Closing all the connections")
	for i := range peerConnectionTcpList {
		peerConnectionTcpList[i].connId.Close()
	}

}
