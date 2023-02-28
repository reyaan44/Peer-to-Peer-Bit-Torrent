package main

import (
	"encoding/binary"
	"fmt"
	"net"
	"net/url"
	"time"

	gotorrentparser "github.com/j-muller/go-torrent-parser"
)

func sendConnectionRequest(buff []byte, Torrent *gotorrentparser.Torrent, peers *[]Peer, current int) {

	defer wg.Done()

	// Parsing the Announce of the Torrent in URL
	URL, err := url.Parse(Torrent.Announce[current])
	if err != nil {
		fmt.Println(err)
		return
	}

	// Establishing a connection by sending a UDP request packet
	connection, err := net.Dial("udp", URL.Host)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer connection.Close()

	// Waiting for 5 seconds for the response
	err = connection.SetReadDeadline(time.Now().Add(5 * time.Second))
	if err != nil {
		fmt.Println(err)
		return
	}

	// Sending the Connection Request Packet
	connection.Write(buff)

	// Reading the Connection
	recieved := make([]byte, 16)
	total_bytes, err := connection.Read(recieved)

	if err != nil {
		fmt.Println(err)
		return
	}
	if total_bytes < 16 {
		fmt.Println("Recieved Bytes in Connection Response is not 16")
		return
	}
	// Checking for Transaction ID
	response_conn := parseConnResponse(recieved)
	if response_conn.transaction_id != binary.BigEndian.Uint32(buff[12:]) {
		panic("Transaction ID not same")
	}

	if response_conn.action == 0 {

		announceBuff := buildAnnounceRequest(response_conn.connId, Torrent)
		// Sending the connection the Announce Packet
		connection.Write(announceBuff)
		// Getting the response
		recieved := make([]byte, 1024)
		no_of_bytes, err := connection.Read(recieved)

		if err != nil {
			fmt.Println(err)
			return
		}
		if no_of_bytes < 20 {
			fmt.Println("Total Bytes smaller than 20 in Announce Response")
			return
		}
		// Parsing the Recieved Data
		response_ann := parseAnnounceRequest(recieved, no_of_bytes)
		// Check for Transaction ID
		if response_ann.transaction_id != binary.BigEndian.Uint32(announceBuff[12:16]) {
			panic("Announce Response Transaction ID not same")
		}
		// Check for action
		if response_ann.action != 1 {
			fmt.Println("Announce Response Action not 1")
		}
		// ... is used when we append a slice to another slice
		*peers = append(*peers, response_ann.PeerList...)

	} else {
		fmt.Println("Response Action is not for Connection")
		return
	}
}

func getUniquePeersList(peersList []Peer) []Peer {

	obj := []Peer{}

	// making a map to find distinct values
	freq := map[Peer]bool{}

	for _, i := range peersList {
		freq[i] = true
	}

	for i := range freq {
		obj = append(obj, i)
	}

	return obj
}

func getPeersList(Torrent *gotorrentparser.Torrent) []Peer {

	// Making a list of objects for storing Peers
	peersList := []Peer{}

	// Building the connection Request
	buff := buildConnRequest()

	// Sending the Connection Request to get the Peers List
	for pos := range Torrent.Announce {
		if Torrent.Announce[pos][0:3] == "udp" {
			wg.Add(1)
			go sendConnectionRequest(buff, Torrent, &peersList, pos)
		}
	}
	wg.Wait()

	peersList = getUniquePeersList(peersList)

	return peersList
}
