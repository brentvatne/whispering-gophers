// Solution to exercise four of the whispering gophers codelab.
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

var master = flag.String("master", "", "master peer address")

func main() {
	flag.Parse()
	if *master == "" {
		log.Println("missing -master")
		flag.Usage()
		return
	}
	go connectToPeer(*master)

	l, err := util.Listen()
	if err != nil {
		log.Fatal(err)
	}
	addr := l.Addr().String()
	log.Println("listen on", addr)

	go listen(l)
	go announce(addr)

	r := bufio.NewReader(os.Stdin)
	for {
		line, err := r.ReadString('\n')
		if l := len(line); l > 1 {
			send(line[:l-1], addr)
		}
		if err != nil {
			log.Fatalf("read from stdin: %v\n", err)
		}
	}
}

// announce sends a message with empty body every 5 seconds
func announce(self string) {
	for _ = range time.Tick(5 * time.Second) {
		send("", self)
	}
}

type Message struct {
	Body string // Message content
	Addr string // Source address
}

// Map of peers with a Mutex for safe concurrent access.
var peers = struct {
	m map[string]chan<- Message
	sync.RWMutex
}{m: make(map[string]chan<- Message)}

// connectToPeer handles the connection with a peer
func connectToPeer(addr string) {
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
}

// send sends a message to all the peers in the peer list.
func send(msg, addr string) {
	m := Message{Body: msg, Addr: addr}
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
			log.Printf("decode message: %v", err)
			break
		}
		go connectToPeer(msg.Addr)
		if len(msg.Body) > 0 {
			log.Println(msg.Body)
		}
	}
}
