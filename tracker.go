package main

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"

	gotorrentparser "github.com/j-muller/go-torrent-parser"
	bencode "github.com/jackpal/bencode-go"
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

func sendHTTPConnectionRequest(Torrent *gotorrentparser.Torrent, peers *[]Peer, current int, reBuild bool) {

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

	params := url.Values{}

	// Adding InfoHash 20 byte string
	infoHash, _ := hex.DecodeString(Torrent.InfoHash)

	// Adding Port
	params.Add("port", "6881")

	// Adding Uploaded
	params.Add("uploaded", strconv.Itoa(uploadedTillNow))

	// Adding Downloaded
	params.Add("downloaded", strconv.Itoa(downloadedTillNow))

	// Adding Left
	params.Add("left", strconv.Itoa(leftTillNow))

	// Adding Compact
	params.Add("compact", "1")

	// Adding Numwant
	params.Add("numwant", "100")

	// Send an HTTP GET request to the tracker with the URL query string
	requestString := "http://" + URL.Host + "/announce?info_hash=" + url.QueryEscape(string(infoHash)) + "&peer_id=" + url.QueryEscape(string(myPeerId)) + "&" + params.Encode()
	fmt.Println(requestString)

	// To obtain the response of the GET request
	response, err := http.Get(requestString)
	if err != nil {
		fmt.Println("Error sending request to tracker:", err)
		return
	}
	defer response.Body.Close()

	// Read response body as bytes
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return
	}

	// Decode bencoded data
	decodedBody, err := bencode.Decode(bytes.NewReader(body))
	if err != nil {
		fmt.Println("Error decoding bencoded data:", err)
		return
	}

	// Extract peers from tracker response
	PeersBytes, ok := decodedBody.(map[string]interface{})["peers"].(string)
	if !ok {
		fmt.Println("Error extracting peers from tracker response")
		return
	}

	// Extract peer IP addresses and ports
	for i := 0; i < len(PeersBytes); i += 6 {

		buff := []byte(PeersBytes[i : i+6])

		peerObj := new(Peer)

		peerObj.IP = net.IP(buff[0:4]).String()
		peerObj.Port = binary.BigEndian.Uint16(buff[4:6])
		peerObj.Handshake = false
		peerObj.InsideQueue = false
		peerObj.PiecesDownload = 0
		peerObj.PiecesUpload = 0

		*peers = append(*peers, *peerObj)

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
			go sendConnectionRequest(buff, Torrent, &peersList, pos, false)
		} else if Torrent.Announce[pos][0:4] == "http" {
			wg.Add(1)
			go sendHTTPConnectionRequest(Torrent, &peersList, pos, false)
		}
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
		} else if Torrent.Announce[pos][0:4] == "http" {
			wgRebuild.Add(1)
			go sendHTTPConnectionRequest(Torrent, &newPeersList, pos, true)
		}
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
