package main

import (
	"net"
)

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

type File struct {
	index  int
	path   string
	length int
}

type Piece struct {
	index       uint32
	length      uint32
	hash        [20]byte
	data        []byte
	filesOffset []struct {
		startOffset  int
		lengthOffset int
		fileOffset   File
	}
}

type PeerConnection struct {
	connId     net.Conn
	peer       *Peer
	peerId     []byte
	choked     bool
	interested bool
	bitfield   []bool
}

type Peer struct {
	IP             string
	Port           uint16
	Handshake      bool
	InsideQueue    bool
	PiecesDownload int // Pieces Downloaded by us
	PiecesUpload   int // Pieces Uploaded by us
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
