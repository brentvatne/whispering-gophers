package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"log"
	"net"
	"os"
	"sync"
	"time"

	"code.google.com/p/whispering-gophers/util"
)

var (
	master     = flag.String("master", "", "Master peer address, if empty this is a master.")
	defaultTTL = flag.Int("ttl", 5, "TTL for the messages, if 0 TTL is ignored.")

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

// Map of seen message IDs with a Mutex for safe concurrent access.
var seenIDs = struct {
	m map[string]bool
	sync.Mutex
}{m: make(map[string]bool)}

// seen marks an message id as seen, and returns true if it had been seen before.
func seen(id string) bool {
	seenIDs.Lock()
	old := seenIDs.m[id]
	seenIDs.m[id] = true
	seenIDs.Unlock()
	return old
}

// Map of peers with a Mutex for safe concurrent access.
var peers = struct {
	m map[string]chan<- Message
	sync.RWMutex
}{m: make(map[string]chan<- Message)}

// connectToPeer handles the connection with a peer
func connectToPeer(addr string) {
	if addr == listenAddr {
		return
	}
	// Don't reconnect if the peer is already connected
	peers.RLock()
	_, ok := peers.m[addr]
	peers.RUnlock()
	if ok {
		return
	}

	// Connect to peer
	c, err := net.Dial("tcp", addr)
	if err != nil {
		log.Println(err)
		return
	}
	defer c.Close()

	ch := make(chan Message)

	// Add the channel to the peer map
	peers.Lock()
	peers.m[addr] = ch
	peers.Unlock()

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

	// Remove the channel from the peer map
	peers.Lock()
	delete(peers.m, addr)
	peers.Unlock()
	log.Println("Disconnected from", addr)
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
	peers.RLock()
	for _, peer := range peers.m {
		go func(p chan<- Message) { p <- m }(peer)
	}
	peers.RUnlock()
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
			log.Println(err)
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
