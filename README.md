# Peer-to-Peer-Bit-Torrent

## How to run
1. git clone https://github.com/reyaan44/Peer-to-Peer-Bit-Torrent.git
2. go get .
3. Download the required torrent file in 'torrents' folder
4. go run . "torrent file name"

Eg :- go run . movie1.torrent


## 20-04-2023
### Till Now
1. The bittorrent client supports single-file and multi-file downloads with efficient distribution capabilities.
2. The client utilizes UDP and HTTP trackers and modern internet protocols to communicate with peers in the network.
3. Network scanning for peer discovery is periodically performed to maintain up-to-date peer lists.
4. Automatic reconnection to previously closed connections is implemented to maximize download success rates.
5. Data is written directly to disk upon receipt of a piece to optimize performance and prevent memory bloat.
6. Support for upload and bad peer detection is included in the client.
7. The client avoids redundant resource usage by skipping pieces that have already been downloaded (A Naive Pause/Resume Functionality).
8. Peer rating system implementation to identify and avoid bad peers.

### Todo
1. Explore DHT.
2. Explore Magnet Links.
3. NAT traversal implementation for improved peer connectivity behind firewalls.
4. Error Handling

## 04-03-2023
### Till Now
1. The bittorrent client supports both single-file and multi-file downloads, allowing users to efficiently download large files in a distributed manner.
2. The client leverages UDP trackers for efficient and reliable communication with peers in the bittorrent network, using the latest internet protocols and technologies.
3. The client utilizes a smart, yet simple algorithm for mapping pieces to connections, providing a seamless and fast experience for users.
4. The client periodically scans the network for new peers to connect with, ensuring that the latest and most up-to-date peer lists are being used.
5. The client establishes new connections with peers via handshakes in set intervals, and can automatically reconnect to connections that previously closed, maximizing the chance of successful downloads.
6. The client writes files directly to disk as soon as data for a piece is received, preventing memory bloat and ensuring optimal performance.
### Todo
1. (DONE) Work on adding support for additional trackers beyond UDP, such as HTTP to provide more options for users to connect with peers.
2. (DONE) Work on improving the file resuming functionality, ensuring that only the missing pieces are downloaded, and the existing pieces are not re-downloaded, optimizing file downloads.
3. (DONE) Implement an estimated time left feature to display the time remaining for download completion, improving user satisfaction and experience.
4. (DONE) Work on implementing uploads to ensure efficient sharing of files in the bittorrent network, optimizing the upload/download ratio for improved performance.
5. (DONE) Design a performance evaluation algorithm for peers, which takes into account factors such as download/upload pieces, latency, and connectivity, to determine optimal upload/download ratios and tradeoffs, providing a smooth and reliable user experience.
6. Add support for magnet links, a popular and efficient way to share files in the bittorrent network, allowing for easy downloads and sharing of content.
7. Implement DHT support to enable peer discovery and connectivity beyond the tracker, improving overall network efficiency and reliability.
8. Work on implementing NAT traversal to enable connectivity between peers behind NAT firewalls, improving the reach and accessibility of the client.

## 02-03-2023
### Till Now
1. The bittorrent client currently supports the download of a single file using the bittorrent protocol.
2. The client currently only supports the use of UDP trackers for communication and coordination between peers.
3. The client uses a naive (brute force) algorithm for mapping pieces to connections between peers.
### Todo
1. (DONE) Implement support for multi-file downloads, enabling users to download multiple files in a torrent simultaneously. 
2. Enhance the client's functionality by incorporating additional tracker protocols such as HTTP, TCP, and WebSockets, in order to broaden the range of supported tracker types.
3. (DONE) Develop a robust upload feature that allows users to share data with other clients in the swarm, and prioritize uploads to maximize overall download speed.
4. (DONE) Implement a re-handshaking mechanism to automatically re-establish TCP connections that have failed after a successful handshake, thereby ensuring the client can continue to participate in the swarm.
5. (DONE) Periodically scan for and connect to new peers to maintain a healthy swarm with high availability and fast download speeds.
