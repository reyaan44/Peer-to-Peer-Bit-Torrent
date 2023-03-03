package main

import (
	"crypto/sha1"
	"fmt"
	"os"
	"sync"

	gotorrentparser "github.com/j-muller/go-torrent-parser"
)

var mutexDataWrite sync.Mutex

func startNewDownload(peerConnection *PeerConnection, Torrent *gotorrentparser.Torrent,
	QueueNeededPieces chan *Piece, QueueFinishedPieces chan *Piece, pieces []*Piece) {

	defer wg.Done()

	SendInterested(peerConnection)
	SendUnchoke(peerConnection)
	StartReadMessage(peerConnection, pieces)

	for {
		select {
		case currPiece, ok := <-QueueNeededPieces:

			if !ok {
				// channel is closed, exit loop
				return
			}

			// process currPiece
			if peerConnection.choked == true {
				QueueNeededPieces <- currPiece
				err := StartReadMessage(peerConnection, pieces)
				if err != nil {
					fmt.Println(err)
					return
				}
				continue
			}
			if peerConnection.bitfield[currPiece.index] == false {
				QueueNeededPieces <- currPiece
				continue
			}

			fmt.Printf("Sending request for piece : %d to connectionId : %d\n", currPiece.index, peerConnection.peerId[:])
			recievedChecked := requestPiece(peerConnection, currPiece.index, pieces)
			if recievedChecked == false {
				QueueNeededPieces <- currPiece
				return
			}

			QueueFinishedPieces <- currPiece
			fmt.Printf("Recieved Piece : %d from connectionId : %d\n", currPiece.index, peerConnection.peerId[:])
			fmt.Printf("Download = %.0f%%\n", float64(len(QueueFinishedPieces)*100)/float64(pieceCount))
			sendHave(peerConnection, currPiece.index)

		default:
			// channel is empty, no more data expected, exit loop
			return

		}
	}
}

func requestPiece(peerConnection *PeerConnection, index uint32, pieces []*Piece) bool {

	for i := 0; i < int(pieces[index].length); i += 16384 {

		blockSize := min(16384, int(pieces[index].length)-i)
		SendRequest(peerConnection, index, i, blockSize)
	}

	err := StartReadMessage(peerConnection, pieces)

	if err != nil {
		fmt.Println("Did Not Recieve Any Data")
		return false
	}

	check := sha1.Sum(pieces[index].data) == pieces[index].hash
	if check == false {
		fmt.Println("Hash does not match")
		return false
	}

	done := writeToDisk(pieces[index])

	// To save memory, we can delete the data from the piece
	pieces[index].data = nil

	return done

}

func writeToDisk(pieces *Piece) bool {

	// Only 1 goroutine can access the shared resource at a time

	wgDataWrite := sync.WaitGroup{}

	currentPieceOffset := 0

	for pos := range pieces.filesOffset {

		currPos := pos
		currCurrentPieceOffset := currentPieceOffset
		wgDataWrite.Add(1)
		go func(currPos int, currCurrentPieceOffset int) {

			// Lock the mutex to ensure exclusive access to the file
			mutexDataWrite.Lock()
			defer mutexDataWrite.Unlock()
			defer wgDataWrite.Done()

			startPiece := pieces.filesOffset[currPos].startOffset // This is offset for file
			length := pieces.filesOffset[currPos].lengthOffset    // This is for file and pieces both
			File := pieces.filesOffset[currPos].fileOffset        // This is file
			data := pieces.data[currCurrentPieceOffset : currCurrentPieceOffset+length]

			// Open the file for writing
			f, err := os.OpenFile(File.path, os.O_RDWR|os.O_CREATE, 0777)
			if err != nil {
				panic(err)
			}
			defer f.Close()

			_, err = f.WriteAt(data[:], int64(startPiece))
			if err != nil {
				panic(err)
			}

		}(currPos, currCurrentPieceOffset)

		currentPieceOffset += pieces.filesOffset[pos].lengthOffset
	}

	wgDataWrite.Wait()

	return true

}

func setFilePieceOffset(pieces []*Piece, fileList []File) {

	currentPiece := 0
	currentOffset := 0

	for _, file := range fileList {

		f, err := os.OpenFile(file.path, os.O_RDWR|os.O_CREATE, 0777)
		if err != nil {
			fmt.Println("Error opening file: ", err)
			return
		}
		defer f.Close()

		err = f.Truncate(int64(file.length))
		if err != nil {
			panic(err)
		}

		len := file.length
		currFilePos := 0

		for len > 0 {

			used := min(len, int(pieces[currentPiece].length)-currentOffset)

			pieces[currentPiece].filesOffset = append(pieces[currentPiece].filesOffset,
				struct {
					startOffset  int
					lengthOffset int
					fileOffset   File
				}{
					currFilePos,
					used,
					file,
				},
			)

			len -= used
			currFilePos += used
			currentOffset += used
			currentPiece += currentOffset / pieceSize
			currentOffset %= pieceSize

		}

	}

}
