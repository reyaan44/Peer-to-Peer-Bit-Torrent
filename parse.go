package main

import (
	"encoding/binary"
	"fmt"
	"net"
)

func parseConnResponse(buff []byte) ConnResp {

	// Creating an object for the struct
	obj := new(ConnResp)

	// First 4 bytes are for action (0 for Connect)
	obj.action = binary.BigEndian.Uint32(buff[:4])

	// Next 4 bytes are for the transaction_id (Random number we sent)
	obj.transaction_id = binary.BigEndian.Uint32(buff[4:8])

	// Next 8 bytes are the Connection Id
	obj.connId = binary.BigEndian.Uint64(buff[8:])

	return *obj
}

func parseAnnounceRequest(buff []byte, n int) AnnResp {

	// Making an object of Announce Response
	obj := new(AnnResp)

	// 1->4 for action
	obj.action = binary.BigEndian.Uint32(buff[:4])

	// 4->8 for transaction id
	obj.transaction_id = binary.BigEndian.Uint32(buff[4:8])

	// 8->12 for interval
	obj.interval = binary.BigEndian.Uint32(buff[8:12])

	// 12->16 for leechers
	obj.leechers = binary.BigEndian.Uint32(buff[12:16])

	// 16->20 for seeders
	obj.seeders = binary.BigEndian.Uint32(buff[16:20])

	// 20->... First 4, Ip, next 2, Port
	for i := 20; i < n; i += 6 {
		peerObj := new(Peer)
		for j := i; j < i+3; j++ {
			peerObj.IP += fmt.Sprintf("%d.", buff[j])
		}
		peerObj.IP += fmt.Sprintf("%d", buff[i+3])
		peerObj.Port = binary.BigEndian.Uint16(buff[i+4 : i+6])
		peerObj.Handshake = false
		peerObj.InsideQueue = false
		peerObj.PiecesDownload = 0
		peerObj.PiecesUpload = 0
		obj.PeerList = append(obj.PeerList, *peerObj)
	}

	return *obj
}

func parseHandShakeResp(buff []byte, connId net.Conn, currentPeer *Peer) PeerConnection {

	// Making a new object of the HandShake Response
	obj := new(PeerConnection)

	obj.connId = connId

	obj.peer = currentPeer

	obj.peerId = buff[48:]

	obj.choked = true

	obj.interested = false

	obj.bitfield = make([]bool, pieceCount)

	return *obj
}
