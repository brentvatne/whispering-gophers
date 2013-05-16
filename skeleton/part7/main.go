// Solution to part 7 of the Whispering Gophers code lab.
//
// This program extends part 5.
//
// It connects to the peer specified by -peer.
// It accepts connections from peers and receives messages from them.
// When it sees a peer with an address it hasn't seen before, it opens a
// connection to that peer.
//
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"sync"

	"code.google.com/p/whispering-gophers/util"
)

var (
	peerAddr = flag.String("peer", "", "peer host:port")
	self     string
)

type Message struct {
	Addr string
	Body string
}

func main() {
	flag.Parse()

	l, err := util.Listen()
	if err != nil {
		log.Fatal(err)
	}
	self = l.Addr().String()
	log.Println("Listening on", self)

	go dial(*peerAddr)
	go readInput()

	for {
		c, err := l.Accept()
		if err != nil {
			log.Fatal(err)
		}
		go serve(c)
	}
}

var peers = &Peers{m: make(map[string]chan<- Message)}

type Peers struct {
	m  map[string]chan<- Message
	mu sync.RWMutex
}

func (p *Peers) Add(addr string) <-chan Message {
	// TODO: Take the write lock on p.mu. Unlock it before returning (using defer).

	// TODO: Check if the address is already in the peers map under the key addr.
	// TODO: If it is, return nil.

	// TODO: Make a new channel of messages
	// TODO: Add it to the peers map
	// TODO: Return the newly created channel.
}

func (p *Peers) Remove(addr string) {
	// TODO: Take the write lock on p.mu. Unlock it before returning (using defer).
	// TODO: Delete the peer from the peers map.
}

func (p *Peers) List() []chan<- Message {
	// TODO: Take the read lock on p.mu. Unlock it before returning (using defer).
	// TODO: Declare a slice of chan<- Message.

	for /* TODO: Iterate over the map using range */ {
		// TODO: Append each channel into the slice.
	}
	// TODO: Return the slice.
}

func broadcast(m Message) {
	for /* TODO: Range over the list of peers */ {
		// TODO: Send a message to the channel, but don't block.
		// Hint: Select is your friend.
	}
}

func serve(c net.Conn) {
	defer c.Close()
	d := json.NewDecoder(c)
	for {
		var m Message
		err := d.Decode(&m)
		if err != nil {
			log.Println(err)
			return
		}

		// TODO: Launch dial in a new goroutine, to connect to the address in the message's Addr field.

		fmt.Printf("%#v\n", m)
	}
}

func readInput() {
	s := bufio.NewScanner(os.Stdin)
	for s.Scan() {
		m := Message{
			Addr: self,
			Body: s.Text(),
		}
		broadcast(m)
	}
	if err := s.Err(); err != nil {
		log.Fatal(err)
	}
}

func dial(addr string) {
	// TODO: Don't try to dial self.

	// TODO: Add the address to the peers map.
	// TODO: If you get a nil channel, the peer is already connected, return.
	// TODO: Otherwise, remove the address from peers using defer.

	c, err := net.Dial("tcp", addr)
	if err != nil {
		log.Println(addr, err)
		return
	}
	defer c.Close()

	e := json.NewEncoder(c)
	for m := range ch {
		err := e.Encode(m)
		if err != nil {
			log.Println(addr, err)
			return
		}
	}
}
