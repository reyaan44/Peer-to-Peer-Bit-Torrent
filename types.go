package main

import (
	"net"
)

type bencodeTorrent struct {
	Announce string `bencode:"announce"`
	Info     struct {
		PieceLength int    `bencode:"piece length"`
		Pieces      string `bencode:"pieces"`
		Name        string `bencode:"name"`
		Length      int    `bencode:"length"`
	} `bencode:"info"`
}

type bencodeTorrentFiles struct {
	Announce string `bencode:"announce"`
	Info     struct {
		Name        string `bencode:"name"`
		PieceLength int    `bencode:"piece length"`
		Pieces      string `bencode:"pieces"`
		Length      int    `bencode:"length,omitempty"`
		Files       []struct {
			Length int      `bencode:"length"`
			Path   []string `bencode:"path"`
		} `bencode:"files,omitempty"`
	} `bencode:"info"`
}

type Piece struct {
	index    uint32
	length   uint32
	hash     [20]byte
	finished bool
	data     []byte
}

type PeerConnection struct {
	connId     net.Conn
	peer       Peer
	peerId     []byte
	choked     bool
	interested bool
	bitfield   []bool
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
