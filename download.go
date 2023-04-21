package main

import (
	"crypto/sha1"
	"fmt"
	"os"
	"sync"
	"time"

	gotorrentparser "github.com/j-muller/go-torrent-parser"
)

var mutexDataWrite sync.Mutex
var averageSpeed float64
var SMOOTHING_FACTOR = 0.5

func startNewDownload(peerConnection *PeerConnection, Torrent *gotorrentparser.Torrent,
	QueueNeededPieces chan *Piece, QueueFinishedPieces chan *Piece, pieces []*Piece) {

	defer wg.Done()

	err := StartReadMessage(peerConnection, pieces)
	if err != nil {
		return
	}
	sendBitfield(peerConnection)

	for {
		select {
		case currPiece, ok := <-QueueNeededPieces:

			if !ok {
				// channel is closed, exit loop
				return
			}

			if peerConnection.peer.Handshake == false || peerConnection.peer.InsideQueue == false {
				QueueNeededPieces <- currPiece
				peerConnection.peer.Handshake = false
				peerConnection.peer.InsideQueue = false
				return
			}

			if peerConnection.choked == true {
				QueueNeededPieces <- currPiece
				err := StartReadMessage(peerConnection, pieces)
				if err != nil {
					fmt.Println(err)
					peerConnection.peer.Handshake = false
					peerConnection.peer.InsideQueue = false
					return
				}
				continue
			}

			if peerConnection.bitfield[currPiece.index] == false {
				QueueNeededPieces <- currPiece
				continue
			}

			startTime := time.Now()
			// process currPiece
			fmt.Printf("Sending request for piece : %d to connectionId : %d\n", currPiece.index, peerConnection.peerId[:])
			recievedChecked := requestPiece(peerConnection, currPiece.index, pieces)
			if recievedChecked == false {
				QueueNeededPieces <- currPiece
				peerConnection.peer.Handshake = false
				peerConnection.peer.InsideQueue = false
				return
			}

			peerConnection.peer.PiecesDownload++
			QueueFinishedPieces <- currPiece
			myBitfield[currPiece.index] = true

			downloadedTillNow += int(currPiece.length)
			leftTillNow -= int(currPiece.length)

			elapsedTime := time.Since(startTime)
			currDownloadSpeed := float64(currPiece.length) / elapsedTime.Seconds()
			averageSpeed = SMOOTHING_FACTOR*currDownloadSpeed + (1-SMOOTHING_FACTOR)*averageSpeed
			timeLeft := time.Duration(float64(leftTillNow) / averageSpeed * float64(time.Second))

			fmt.Printf("Recieved Piece : %d from connectionId : %d\n", currPiece.index, peerConnection.peerId[:])
			fmt.Printf("Download = %.2f%%\n", float64(downloadedTillNow*100)/float64(leftTillNow+downloadedTillNow))
			fmt.Printf("Downloaded = %.2f MB / %.2f MB\n", float64(downloadedTillNow)/float64(1024*1024), float64(leftTillNow+downloadedTillNow)/float64(1024*1024))
			fmt.Printf("Download Speed = %.2f KB/s\n", averageSpeed/float64(1024))
			fmt.Printf("Time Left = %s\n", timeLeft.Round(time.Second).String())

			// Check if peer is bad, if yes, close the connection
			PiecesDownload := peerConnection.peer.PiecesDownload
			PiecesUpload := peerConnection.peer.PiecesUpload

			if PiecesDownload >= 5 {
				if PiecesUpload == 0 {
					PiecesUpload = 1
				}
				ratio := float64(PiecesDownload) / float64(PiecesUpload)
				if ratio >= 1 {
					fmt.Println("Good Peer : ", peerConnection.peerId, " Ratio : ", ratio)
					SendUnchoke(peerConnection)
				} else if ratio < 0.01 {
					fmt.Println("Bad Peer : ", peerConnection.peerId, " Ratio : ", ratio)
					peerConnection.peer.Handshake = false
					peerConnection.peer.InsideQueue = false
					SendChoke(peerConnection)
					return
				}
			}

			err := StartReadMessage(peerConnection, pieces)
			if err != nil {
				return
			}

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

	// TODOL Handle Panic calls here

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

func readFromDisk(pieces *Piece) ([]byte, bool) {

	// Only 1 goroutine can access the shared resource at a time

	wgDataRead := sync.WaitGroup{}

	currentPieceOffset := 0
	data := make([]byte, pieces.length)

	for pos := range pieces.filesOffset {

		currPos := pos
		currCurrentPieceOffset := currentPieceOffset
		wgDataRead.Add(1)
		go func(currPos int, currCurrentPieceOffset int) {

			// Lock the mutex to ensure exclusive access to the file
			mutexDataWrite.Lock()
			defer mutexDataWrite.Unlock()
			defer wgDataRead.Done()

			startPiece := pieces.filesOffset[currPos].startOffset // This is offset for file
			length := pieces.filesOffset[currPos].lengthOffset    // This is for file and pieces both
			File := pieces.filesOffset[currPos].fileOffset        // This is file

			// Open the file for writing
			f, err := os.OpenFile(File.path, os.O_RDWR|os.O_CREATE, 0777)
			if err != nil {
				return
			}
			defer f.Close()

			_, err = f.ReadAt(data[currCurrentPieceOffset:currCurrentPieceOffset+length], int64(startPiece))
			if err != nil {
				return
			}

		}(currPos, currCurrentPieceOffset)

		currentPieceOffset += pieces.filesOffset[pos].lengthOffset
	}

	wgDataRead.Wait()

	check := sha1.Sum(data[:]) == pieces.hash
	if check == false {
		fmt.Println("Read, Data Hash does not Match")
		return data, false
	} else {
		fmt.Println("Read, Data Hash Matched")
	}

	return data, true

}

func setFilePieceOffset(pieces []*Piece, fileList []File) {

	currentPiece := 0
	currentOffset := 0

	for _, file := range fileList {

		f, err := os.OpenFile(file.path, os.O_RDWR|os.O_CREATE, 0777)
		if err != nil {
			panic(err)
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
