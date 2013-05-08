package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"log"
	"net"
	"os"
	"time"

	"code.google.com/p/whispering-gophers/util"
)

var (
	master     = flag.String("master", "", "Master peer address, if empty this is a master.")
	defaultTTL = flag.Int("ttl", 5, "TTL for the messages, if 0 TTL is ignored.")
	checkSeen  = flag.Bool("seen", true, "check for already seen messages and discard them.")

	// Address sent in messages created by this peer.
	listenAddr string
)

func main() {
	flag.Parse()
	if *master != "" {
		go connectToPeer(*master)
	}

	l, err := util.Listen()
	if err != nil {
		log.Fatal(err)
	}
	listenAddr = l.Addr().String()
	log.Println("listen on", listenAddr)

	go listen(l)
	go announce()
	go seenMonitor()
	go peersMonitor()

	r := bufio.NewReader(os.Stdin)
	for {
		line, err := r.ReadString('\n')
		if l := len(line); l > 1 {
			send(Message{Body: line[:l-1]})
		}
		if err != nil {
			log.Fatalf("read from stdin: %v\n", err)
		}
	}
}

// announce sends an empty message every 5 seconds
func announce() {
	for _ = range time.Tick(5 * time.Second) {
		send(Message{})
	}
}

type Message struct {
	Body string // Message content
	Addr string // Source address
	ID   string // Message ID, should be unique
	TTL  int    // Remaining TTL
}

// seenRequest is a request to the seen monitor to mark an id as seen and
// return a boolean indicating whether the id had been seen before in the
// passed channel result.
type seenRequest struct {
	id     string
	result chan bool
}

var seenChan = make(chan seenRequest)

// seenMonitor reads and processes seenRequests from the seenChan channel.
func seenMonitor() {
	seenIDs := make(map[string]bool)
	for req := range seenChan {
		old := seenIDs[req.id]
		seenIDs[req.id] = true
		req.result <- old
	}
}

// seen marks an message id as seen, and returns true if it had been seen before.
func seen(id string) bool {
	if !*checkSeen {
		return false
	}
	r := make(chan bool)
	seenChan <- seenRequest{id, r}
	return <-r
}

// List returns a slice of the channels in the peer list.
func ListPeers() []chan<- Message {
	res := make(chan []chan<- Message)
	peers <- listPeersReq{res}
	return <-res
}

// listPeersReq is a request to send a slice containing the list of peers into
// the passed channel result.
type listPeersReq struct {
	result chan []chan<- Message
}

// Add creates and returns a new channel that is also added to the peer
// list if the given address wasn't already in the peer list.
// If it was, no channel is created and Add returns nil.
func AddPeer(addr string) <-chan Message {
	res := make(chan chan Message)
	peers <- addPeerReq{addr, res}
	return <-res
}

// addPeerReq is a request to create a new channel for a peer given its address
// if it wasn't already in the peer list.
// The new channel is returned in the given channel result.
type addPeerReq struct {
	addr   string
	result chan chan Message
}

// Remove removes a peer from the peer list given its address.
func RemovePeer(addr string) {
	peers <- rmPeerReq{addr}
}

// rmPeerReq is the request to remove a peer from the peer list.
type rmPeerReq struct {
	addr string
}

var peers = make(chan interface{})

func peersMonitor() {
	peersList := make(map[string]chan<- Message)
	for req := range peers {
		switch req := req.(type) {
		case listPeersReq:
			l := make([]chan<- Message, 0, len(peers))
			for _, p := range peersList {
				l = append(l, p)
			}
			req.result <- l
		case addPeerReq:
			if _, ok := peersList[req.addr]; ok {
				close(req.result)
				break
			}
			ch := make(chan Message)
			peersList[req.addr] = ch
			req.result <- ch
		case rmPeerReq:
			delete(peersList, req.addr)
		}
	}
}

// connectToPeer handles the connection with a peer
func connectToPeer(addr string) {
	ch := AddPeer(addr)
	if ch == nil {
		return // addr already handled by another goroutine
	}
	defer RemovePeer(addr)

	// Connect to peer
	c, err := net.Dial("tcp", addr)
	if err != nil {
		log.Println(err)
		return
	}
	defer c.Close()

	log.Println("Connected to", addr)

	// Receive messages from the channel and send them to the peer.
	enc := json.NewEncoder(c)
	for m := range ch {
		err := enc.Encode(m)
		if err != nil {
			log.Printf("%v failed: %v\n", addr, err)
			break
		}
	}
}

// send sends a message to all the peers in the peer list, before sending the message
// it initializes non initalized fields with default values.
func send(m Message) {
	if m.TTL == 0 {
		m.TTL = *defaultTTL
	}
	if m.ID == "" {
		m.ID = util.RandomID()
		seen(m.ID)
	}
	if m.Addr == "" {
		m.Addr = listenAddr
	}
	for _, peer := range ListPeers() {
		select {
		case peer <- m:
		default:
			// it's ok to drop the occasional message
			// (better than getting blocked by a slow writer, anyway)
			// reliable delivery is an exercise for the reader
		}
	}
}

// listen accepts new connection from a listener
func listen(l net.Listener) {
	for {
		c, err := l.Accept()
		if err != nil {
			log.Printf("accept new connection: %v", err)
			continue
		}
		go handle(c)
	}
}

// handle handles a connection reading messages from it and
// decoding them to JSON to be processed.
func handle(c net.Conn) {
	dec := json.NewDecoder(c)
	var msg Message
	for {
		err := dec.Decode(&msg)
		if err != nil {
			log.Printf("from %v: %v\n", c.RemoteAddr(), err)
			break
		}
		processMsg(msg)
	}
}

// processMsg handles a received message, prints it, adds the address
// to the peer list if it is new, and broadcasts the message if TTL is
// not zero.
func processMsg(msg Message) {
	if seen(msg.ID) {
		return
	}
	if len(msg.Body) > 0 {
		log.Println(msg.Body)
	}
	go connectToPeer(msg.Addr)

	if *defaultTTL > 0 {
		msg.TTL--
		if msg.TTL <= 0 {
			return
		}
	}
	send(msg)
}
