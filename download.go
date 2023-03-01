package main

import (
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"time"

	gotorrentparser "github.com/j-muller/go-torrent-parser"
)

func messageType(peerConnection *PeerConnection) (int32, []byte, error) {

	// first thing is length, next thing is id

	connection := peerConnection.connId

	err := connection.SetReadDeadline(time.Now().Add(5 * time.Second))
	if err != nil {
		fmt.Println(err)
		return -1, nil, err
	}
	defer connection.SetReadDeadline(time.Time{})

	// Get the length
	length := make([]byte, 4)
	_, err = io.ReadFull(connection, length)
	if err != nil {
		fmt.Println(err)
		return -1, nil, err
	}

	// Keep-Alive Message
	if binary.BigEndian.Uint32(length[:]) == 0 {
		return 0, nil, nil
	}

	// Check for the id
	messageId := make([]byte, 1)
	_, err = io.ReadFull(connection, messageId)
	if err != nil {
		fmt.Println(err)
		return -1, nil, err
	}

	// Get the real message
	lenInteger := int32(binary.BigEndian.Uint32(length))
	lenInteger--
	if lenInteger <= 0 {
		return int32(messageId[0]), nil, nil
	}

	buff := make([]byte, lenInteger)
	_, err = io.ReadFull(connection, buff)
	if err != nil {
		fmt.Println(err)
		return -1, nil, err
	}

	return int32(messageId[0]), buff, nil
}

func messageHandler(peerConnection *PeerConnection, pieces []*Piece) (bool, error) {

	messageId, msg, err := messageType(peerConnection)

	if err != nil {
		fmt.Println(err)
		return false, err
	}
	// TODO: Handle the messages

	switch messageId {
	case 0:
		// Choke Message
		peerConnection.choked = true
		fmt.Println("Choke Message")
	case 1:
		// Unchoke Message
		peerConnection.choked = false
		fmt.Println("Unchoke Message")
	case 2:
		// Interested Message
		peerConnection.interested = true
		fmt.Println("Interested Message")
	case 3:
		// Not Interested Message
		peerConnection.interested = false
		fmt.Println("Not Interested Message")
	case 4:
		// Have Message
		fmt.Println("Have Message")
		peerConnection.bitfield[binary.BigEndian.Uint32(msg)] = true
	case 5:
		// Bitfield Message
		fmt.Println("Bitfield Message")
		currIdx := 0
		for _, val := range msg {
			for i := 0; i < 8 && (currIdx) < len(peerConnection.bitfield); i++ {
				if (val & (1 << int(7-i))) != 0 {
					peerConnection.bitfield[currIdx] = true
				} else {
					peerConnection.bitfield[currIdx] = false
				}
				currIdx++
			}
		}
	case 6:
		// Request Block Message
		fmt.Println("Request Block Message")
	case 7:
		// Send Piece Message
		index := int32(binary.BigEndian.Uint32(msg[0:4]))
		offset := int32(binary.BigEndian.Uint32(msg[4:8]))
		copy(pieces[index].data[offset:], msg[8:])
		fmt.Println("Send Piece Message")
	case 8:
		// Cancel Block Message
		fmt.Println("Cancel Block Message")
	case 9:
		// Port Message
		fmt.Println("Port Message")
	default:
		fmt.Println("Unknown Message")
		return false, nil
	}

	return true, nil
}

func StartReadMessage(peerConnection *PeerConnection, pieces []*Piece) error {

	// If we recieve 0 messages, return error, else, return nil

	totalRecieved := 0

	for {

		check, err := messageHandler(peerConnection, pieces)

		if check == true {
			totalRecieved++
		} else {
			if totalRecieved == 0 {
				return err
			} else {
				return nil
			}
		}

	}
}

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

	return true

}

func SendRequest(peerConnection *PeerConnection, pieceIndex uint32, offset int, length int) {
	buff := buildRequestBlock(pieceIndex, uint32(length), uint32(offset))
	peerConnection.connId.Write(buff)
}

func SendHandshake(currentPeer Peer, Torrent *gotorrentparser.Torrent, peerConnectionList *[]PeerConnection) {

	defer wg.Done()

	// TODO : Unsure if I need to add dots in the Ip address or not
	connection, err := net.Dial("tcp", fmt.Sprintf("%s:%d", currentPeer.IP, currentPeer.Port))
	if err != nil {
		fmt.Println(err)
		return
	}

	// Waiting for 15 seconds for the response
	err = connection.SetReadDeadline(time.Now().Add(15 * time.Second))
	defer connection.SetReadDeadline(time.Time{})
	if err != nil {
		fmt.Println(err)
		connection.Close()
		return
	}

	handshake := buildHandshake(Torrent)
	connection.Write(handshake)

	recieved := make([]byte, 68)
	total_bytes, err := connection.Read(recieved)

	if err != nil {
		fmt.Println(err)
		connection.Close()
		return
	}

	if total_bytes != 68 {
		fmt.Println("Handshake failed, recieved less than 68 bytes")
		connection.Close()
		return
	}

	// Match the pstr
	if string(recieved[1:20]) != string(handshake[1:20]) {
		fmt.Println("Handshake failed, pstr not matched")
		connection.Close()
		return
	}

	// Match the info_hash
	if string(recieved[28:48]) != string(handshake[28:48]) {
		fmt.Println("Handshake failed, info_hash not matched")
		connection.Close()
		return
	}

	// TODO: Pass the total number of pieces to make a bitfield
	response := parseHandShakeResp(recieved, connection, currentPeer)
	if peerConnectionList != nil {
		*peerConnectionList = append(*peerConnectionList, response)
	}
}

func SendInterested(peerConnection *PeerConnection) {

	// Sending the Interested Message
	fmt.Println("Sent Interested")
	peerConnection.connId.Write(buildInterested())

}

func SendUnchoke(peerConnection *PeerConnection) {

	// Sending the Unchoke Message
	fmt.Println("Sent Unchoke")
	peerConnection.connId.Write(buildUnchoke())

}

func sendAlive(peerConnection *PeerConnection) {

	// Sending the Alive Message
	peerConnection.connId.Write(buildKeepAlive())

}

func sendHave(peerConnection *PeerConnection, pieceIndex uint32) {

	// Sending the Have Message
	fmt.Println("Sent Have")
	peerConnection.connId.Write(buildHave(pieceIndex))

}
