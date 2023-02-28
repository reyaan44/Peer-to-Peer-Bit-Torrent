package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"time"

	gotorrentparser "github.com/j-muller/go-torrent-parser"
)

func messageType(peerConnection PeerConnection) (int32, int32, []byte, error) {

	// first thing is length, next thing is id

	connection := peerConnection.connId

	err := connection.SetReadDeadline(time.Now().Add(2 * time.Second))
	if err != nil {
		fmt.Println(err)
		return -1, -1, nil, err
	}

	defer connection.SetReadDeadline(time.Time{})

	// Get the length
	length := make([]byte, 4)
	_, err = io.ReadFull(connection, length)
	if err != nil {
		fmt.Println(err)
		return -1, -1, nil, err
	}

	// Keep-Alive Message
	if binary.BigEndian.Uint32(length) == 0 {
		return 0, 0, nil, nil
	}

	// Check for the id
	messageId := make([]byte, 1)
	_, err = io.ReadFull(connection, messageId)
	if err != nil {
		fmt.Println(err)
		return -1, -1, nil, err
	}

	// Get the real message
	buff := make([]byte, binary.BigEndian.Uint32(length)-1)
	_, err = io.ReadFull(connection, buff)
	if err != nil {
		fmt.Println(err)
		return -1, -1, nil, err
	}

	return int32(length[0]), int32(messageId[0]), buff, nil
}

func messageHandler(peerConnection PeerConnection) (bool, error) {

	length, messageId, msg, err := messageType(peerConnection)

	if err != nil {
		fmt.Println(err)
		return false, err
	}
	// TODO: Handle the messages
	// TODO: Remove the next 3 lines, just so error is not shown
	length = length + 1
	length = length - 1
	fmt.Println("Message : ", msg)

	switch messageId {
	case 0:
		// Choke Message
		fmt.Println("Choke Message")
	case 1:
		// Unchoke Message
		fmt.Println("Unchoke Message")
	case 2:
		// Interested Message
		fmt.Println("Interested Message")
	case 3:
		// Not Interested Message
		fmt.Println("Not Interested Message")
	case 4:
		// Have Message
		fmt.Println("Have Message")
	case 5:
		// Bitfield Message
		fmt.Println("Bitfield Message")
	case 6:
		// Request Block Message
		fmt.Println("Request Block Message")
	case 7:
		// Send Piece Message
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

func StartReadMessage(peerConnection PeerConnection) {

	for {

		check, err := messageHandler(peerConnection)

		if err != nil {
			fmt.Println(err)
			return
		}

		if check == false {
			fmt.Println("No more messages to read")
			return
		}

	}
}

func startNewDownload(peerConnection PeerConnection, Torrent *gotorrentparser.Torrent) {

	defer wg.Done()

	SendInterested(peerConnection)
	SendUnchoke(peerConnection)
	StartReadMessage(peerConnection)

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

func SendInterested(peerConnection PeerConnection) {

	// Sending the Interested Message
	peerConnection.connId.Write(buildInterested())

}

func SendUnchoke(peerConnection PeerConnection) {

	// Sending the Unchoke Message
	peerConnection.connId.Write(buildUnchoke())

}

func sendAlive(peerConnection PeerConnection) {

	// Sending the Alive Message
	peerConnection.connId.Write(buildKeepAlive())

}
