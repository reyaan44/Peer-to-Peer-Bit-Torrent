package main

import "net"

type bencodeTorrent struct {
	Announce string `bencode:"announce"`
	Info     struct {
		PieceLength int    `bencode:"piece length"`
		Pieces      string `bencode:"pieces"`
		Name        string `bencode:"name"`
		Length      int    `bencode:"length"`
	} `bencode:"info"`
}

type PeerConnection struct {
	connId     net.Conn
	peer       Peer
	peerId     []byte
	choked     bool
	interested bool
	bitfield   []byte
}

type Peer struct {
	IP   string
	Port uint16
}

type ConnResp struct {
	action         uint32
	transaction_id uint32
	connId         uint64
}

type AnnResp struct {
	action         uint32
	transaction_id uint32
	interval       uint32
	leechers       uint32
	seeders        uint32
	PeerList       []Peer
}
