package main

import (
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

	err := connection.SetReadDeadline(time.Now().Add(10 * time.Second))
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

	switch messageId {
	case 0:
		// Choke Message
		peerConnection.choked = true
		fmt.Println("Recieved Choke Message")
	case 1:
		// Unchoke Message
		peerConnection.choked = false
		fmt.Println("Recieved Unchoke Message")
	case 2:
		// Interested Message
		peerConnection.interested = true
		fmt.Println("Recieved Interested Message")
	case 3:
		// Not Interested Message
		peerConnection.interested = false
		fmt.Println("Recieved Not Interested Message")
	case 4:
		// Have Message
		fmt.Println("Recieved Have Message")
		peerConnection.bitfield[binary.BigEndian.Uint32(msg)] = true
	case 5:
		// Bitfield Message
		fmt.Println("Recieved Bitfield Message")
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
		// Request Block Message (Peer has requested Block from me)
		fmt.Println("Recieved Message To Send Piece")
		if len(msg) != 12 {
			fmt.Println("Invalid Request Message")
			return false, nil
		}
		index := uint32(binary.BigEndian.Uint32(msg[0:4]))
		begin := uint32(binary.BigEndian.Uint32(msg[4:8]))
		length := uint32(binary.BigEndian.Uint32(msg[8:12]))
		if myBitfield[index] == true {
			buff, ok := readFromDisk(pieces[index])
			if ok == true {
				sendPiece(peerConnection, index, begin, buff[begin:begin+length])
				peerConnection.peer.PiecesUpload++
				uploadedTillNow += int(length)
			}
		}
		// Check if peer is bad, if yes, close the connection
		PiecesDownload := peerConnection.peer.PiecesDownload
		PiecesUpload := peerConnection.peer.PiecesUpload
		if PiecesUpload >= 10 {
			ratio := float64(PiecesDownload) / float64(PiecesUpload)
			if ratio <= 0.5 {
				fmt.Println("Removing Bad Peer : ", peerConnection.peerId, " Ratio : ", ratio)
				SendChoke(peerConnection)
			}
		}
	case 7:
		// Send Piece Message (Peer has sent me a piece)
		fmt.Println("Recieved Piece Message")
		index := int32(binary.BigEndian.Uint32(msg[0:4]))
		offset := int32(binary.BigEndian.Uint32(msg[4:8]))
		if pieces[index].data == nil {
			pieces[index].data = make([]byte, pieces[index].length)
		}
		copy(pieces[index].data[offset:], msg[8:])
	case 8:
		// Cancel Block Message
		fmt.Println("Recieved Cancel Block Message")
	case 9:
		// Port Message
		fmt.Println("Recieved Port Message")
	default:
		fmt.Println("Recieved Unknown Message")
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

func SendRequest(peerConnection *PeerConnection, pieceIndex uint32, offset int, length int) {
	buff := buildRequestBlock(pieceIndex, uint32(length), uint32(offset))
	peerConnection.connId.Write(buff)
}

func SendHandshake(currentPeer *Peer, Torrent *gotorrentparser.Torrent, peerConnectionList *[]PeerConnection,
	QueueNeededPieces chan *Piece, QueueFinishedPieces chan *Piece, pieces []*Piece) {

	defer wg.Done()

	connection, err := net.Dial("tcp", fmt.Sprintf("%s:%d", currentPeer.IP, currentPeer.Port))
	if err != nil {
		fmt.Println(err)
		return
	}

	// Waiting for 10 seconds for the response
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

	currentPeer.Handshake = true
	currentPeer.InsideQueue = true

	fmt.Println("Handshake Successfull with Peer : ", currentPeer.IP, ":", currentPeer.Port)

	response := parseHandShakeResp(recieved, connection, currentPeer)

	if peerConnectionList != nil {
		*peerConnectionList = append(*peerConnectionList, response)
	}

	SendUnchoke(&response)
	SendInterested(&response)
	sendBitfield(&response)

	wg.Add(1)
	go startNewDownload(&response, Torrent, QueueNeededPieces, QueueFinishedPieces, pieces)

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

func SendChoke(peerConnection *PeerConnection) {

	// Sending the Choke Message
	fmt.Println("Sent Choke")
	peerConnection.connId.Write(buildChoke())

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

func sendPiece(peerConnection *PeerConnection, pieceIndex uint32, offset uint32, data []byte) {

	// Sending the Piece Message
	fmt.Println("Sent Piece")
	peerConnection.connId.Write(buildPiecetoSend(pieceIndex, offset, data))

}

func sendBitfield(PeerConnection *PeerConnection) {

	// Sending the Bitfield Message
	fmt.Println("Sent Bitfield")
	PeerConnection.connId.Write(buildBitfield(myBitfield))

}
