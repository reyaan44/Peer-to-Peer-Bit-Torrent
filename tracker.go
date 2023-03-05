package main

import (
	"encoding/binary"
	"fmt"
	"net"
	"net/url"
	"time"

	gotorrentparser "github.com/j-muller/go-torrent-parser"
)

func sendConnectionRequest(buff []byte, Torrent *gotorrentparser.Torrent, peers *[]Peer, current int, reBuild bool) {

	if reBuild == false {
		defer wg.Done()
	} else {
		defer wgRebuild.Done()
	}

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
	defer connection.SetReadDeadline(time.Time{})
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

// func sendHTTPConnectionRequest(Torrent *gotorrentparser.Torrent, peers *[]Peer, current int, reBuild bool) {
// 	if reBuild == false {
// 		defer wg.Done()
// 	} else {
// 		defer wgRebuild.Done()
// 	}

// 	// Parsing the Announce of the Torrent in URL
// 	URL, err := url.Parse(Torrent.Announce[current])
// 	if err != nil {
// 		fmt.Println(err)
// 		return
// 	}

// 	params := url.Values{}

// 	// Adding InfoHash 20 byte string
// 	infoHash, _ := hex.DecodeString(Torrent.InfoHash)
// 	params.Set("info_hash", hex.EncodeToString(infoHash))

// 	// Adding PeerID 20 byte string
// 	params.Set("peer_id", hex.EncodeToString(myPeerId[:]))

// 	// Adding Port
// 	params.Set("port", "6881")

// 	// Adding Uploaded
// 	params.Set("uploaded", strconv.Itoa(uploadedTillNow))

// 	// Adding Downloaded
// 	params.Set("downloaded", strconv.Itoa(downloadedTillNow))

// 	// Adding Left
// 	params.Set("left", strconv.Itoa(leftTillNow))

// 	// Adding Compact
// 	params.Set("compact", "1")

// 	// Send an HTTP GET request to the tracker with the URL query string
// 	requestString := URL.String() + "?" + params.Encode()
// 	fmt.Println("REQUEST = ", requestString)
// 	resp, err := http.Get(requestString)
// 	if err != nil {
// 		fmt.Println("Error sending request to tracker:", err)
// 		return
// 	}

// 	fmt.Println("SENT = ", requestString)

// 	result, err := bencode.Decode(resp.Body)
// 	if err != nil {
// 		fmt.Println("Error decoding tracker response:", err)
// 		return
// 	}

// 	fmt.Println("RESULT = ", result)

// 	defer resp.Body.Close()

// }

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
			go sendConnectionRequest(buff, Torrent, &peersList, pos, false)
		}
		// else if Torrent.Announce[pos][0:4] == "http" {
		// 	wg.Add(1)
		// 	go sendHTTPConnectionRequest(Torrent, &peersList, pos, false)
		// }
	}
	wg.Wait()

	peersList = getUniquePeersList(peersList)

	return peersList
}

func reBuildGetPeersList(Torrent *gotorrentparser.Torrent, peersList *[]Peer) int {

	prevLength := len(*peersList)

	newPeersList := []Peer{}

	// Building the connection Request
	buff := buildConnRequest()

	// Sending the Connection Request to get the Peers List
	for pos := range Torrent.Announce {
		if Torrent.Announce[pos][0:3] == "udp" {
			wgRebuild.Add(1)
			go sendConnectionRequest(buff, Torrent, &newPeersList, pos, true)
		}
		// else if Torrent.Announce[pos][0:4] == "http" {
		// 	wgRebuild.Add(1)
		// 	go sendHTTPConnectionRequest(Torrent, &newPeersList, pos, true)
		// }
	}
	wgRebuild.Wait()

	// making a map to find distinct values
	freq := map[Peer]bool{}

	for _, i := range *peersList {
		freq[i] = true
	}

	for _, i := range newPeersList {
		if _, ok := freq[i]; !ok {
			*peersList = append(*peersList, i)
		}
	}

	return prevLength
}
