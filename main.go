package main

import (
	"fmt"
	"os"
	"sync"
	"time"

	excelize "github.com/360EntSecGroup-Skylar/excelize"
	gotorrentparser "github.com/j-muller/go-torrent-parser"
	bencode "github.com/jackpal/bencode-go"
)

var wg = sync.WaitGroup{}
var wgRebuild = sync.WaitGroup{}

var myPeerId []byte

var pieceCount int
var pieceSize int

var downloadedTillNow int
var uploadedTillNow int
var leftTillNow int

var myBitfield []bool

func main() {

	totalPeersUsedForPieces = make(map[string]bool)

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

	myBitfield = make([]bool, pieceCount)

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

	// Make channels and add Pieces
	QueueNeededPieces := make(chan *Piece, pieceCount)
	QueueFinishedPieces := make(chan *Piece, pieceCount)
	defer close(QueueNeededPieces)
	defer close(QueueFinishedPieces)

	for i := range pieces {
		_, check := readFromDisk(pieces[i])
		if check == true {
			myBitfield[i] = true
			downloadedTillNow += int(pieces[i].length)
			continue
		}
		leftTillNow += int(pieces[i].length)
		QueueNeededPieces <- pieces[i]
	}

	// Parsing the torrent using gotorrentparser
	Torrent, err := gotorrentparser.ParseFromFile(filePath)
	if err != nil {
		panic(err)
	}

	peersList := getPeersList(Torrent)
	fmt.Println("Total Number of Peers : ", len(peersList))

	// For every peer in the list, start a new goroutine for tcp connection
	peerConnectionTcpList := []PeerConnection{}
	for current := range peersList {
		wg.Add(1)
		go SendHandshake(&peersList[current], Torrent, &peerConnectionTcpList, QueueNeededPieces, QueueFinishedPieces, pieces)
	}

	// Create a channel to signal when all downloads are done
	done := make(chan bool)
	// Create a channel to signal when we need to send a ReConnection Message
	channelReConnection := make(chan bool)

	go func() {
		wg.Wait() // Wait for all downloads to complete
		if downloadedTillNow == totalLength && leftTillNow == 0 {
			fmt.Println("Download Finished")
		} else {
			fmt.Println("Some Error, Download Not Finished")
		}
		done <- true // Send a message on the done channel when all downloads have finished
	}()

	go func(channelReConnection chan bool) {
		for {
			time.Sleep(300 * time.Second)
			channelReConnection <- true
		}
	}(channelReConnection)

downloadLoop:
	for {
		select {

		case <-channelReConnection:

			fmt.Println("ReHandshaking and Reconnecting")

			reBuildGetPeersList(Torrent, &peersList)

			fmt.Println("Total Number of Peers Currently : ", len(peersList))

			// For every peer in the list, start a new goroutine for tcp connection
			for current := range peersList {
				if peersList[current].Handshake == true {
					continue
				}
				go SendHandshake(&peersList[current], Torrent, &peerConnectionTcpList, QueueNeededPieces, QueueFinishedPieces, pieces)
			}

		case <-done:
			// All downloads have finished
			break downloadLoop
		}
	}
	// Interesting Case, If all the pieces are downloaded, but then just before that, rebuild is called, it will still send a startDownload message
	wg.Wait()

	fmt.Println("Total Peers Used To Recieve Pieces : ", len(totalPeersUsedForPieces))

	// Create a new Excel file.
	excelFile := excelize.NewFile()

	// Close all the connections
	fmt.Println("Creating Sheet and Closing all the connections")

	// Create a slice of data for Sheet.
	ExcelData := [][]interface{}{
		{"Ip Address", "Port Number", "Pieces Downloaded By Us", "Pieces Uploaded By Us", "Rating"},
	}

	// Normalization
	minRatio := -1.0
	maxRatio := -1.0

	for i := range peerConnectionTcpList {

		// Pieces Downloaded by us
		peerPiecesDownloaded := peerConnectionTcpList[i].peer.PiecesDownload
		// Pieces Uploaded by us
		peerPiecesGiven := peerConnectionTcpList[i].peer.PiecesUpload + 1
		// Ratio
		ratioPieces := float64(peerPiecesDownloaded) / float64(peerPiecesGiven)

		if minRatio == -1 {
			minRatio = ratioPieces
			maxRatio = ratioPieces
		}
		if ratioPieces < minRatio {
			minRatio = ratioPieces
		}
		if ratioPieces > maxRatio {
			maxRatio = ratioPieces
		}
	}

	for i := range peerConnectionTcpList {

		peerConnectionTcpList[i].connId.Close()

		// Pieces Downloaded by us
		peerPiecesDownloaded := peerConnectionTcpList[i].peer.PiecesDownload
		// Pieces Uploaded by us
		peerPiecesGiven := peerConnectionTcpList[i].peer.PiecesUpload + 1
		// Ratio
		ratioPieces := float64(peerPiecesDownloaded) / float64(peerPiecesGiven)

		// Normalization
		rating := (float64(ratioPieces-minRatio) / float64(maxRatio-minRatio+1)) * 100

		ExcelData = append(ExcelData, []interface{}{peerConnectionTcpList[i].peer.IP, peerConnectionTcpList[i].peer.Port,
			peerPiecesDownloaded, peerPiecesGiven, rating})

	}

	// Write the ExcelData to the Excel file.
	for i, row := range ExcelData {
		for j, val := range row {
			cell := excelize.ToAlphaString(j+1) + fmt.Sprintf("%d", i+1)
			excelFile.SetCellValue("Sheet1", cell, val)
		}
	}

	// Save the file.
	if err := excelFile.SaveAs("./Reyaan-Downloads/Reyaan-Peers-Data.xlsx"); err != nil {
		fmt.Println(err)
	}

}
