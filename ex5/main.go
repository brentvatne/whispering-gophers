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
	if !*checkSeen {
		return false
	}
	seenIDs.Lock()
	old := seenIDs.m[id]
	seenIDs.m[id] = true
	seenIDs.Unlock()
	return old
}

var peers = Peers{m: make(map[string]chan<- Message)}

type Peers struct {
	mu sync.RWMutex
	m  map[string]chan<- Message
}

// Add creates and returns a new channel that is also added to the peer
// list if the given address wasn't already in the peer list.
// If it was, no channel is created and Add returns nil.
func (p Peers) Add(addr string) <-chan Message {
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, ok := p.m[addr]; ok {
		return nil
	}
	ch := make(chan Message)
	p.m[addr] = ch
	return ch
}

// Remove removes a peer from the peer list given its address.
func (p Peers) Remove(addr string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.m, addr)
}

// List returns a slice of the channels in the peer list.
func (p Peers) List() []chan<- Message {
	p.mu.RLock()
	defer p.mu.RUnlock()
	s := make([]chan<- Message, 0, len(p.m))
	for _, ch := range p.m {
		s = append(s, ch)
	}
	return s
}

// connectToPeer handles the connection with a peer
func connectToPeer(addr string) {
	ch := peers.Add(addr)
	if ch == nil {
		return // addr already handled by another goroutine
	}
	defer peers.Remove(addr)

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
	for _, peer := range peers.List() {
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
	if msg.Addr == listenAddr {
		return
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
