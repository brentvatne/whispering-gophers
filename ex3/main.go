// Solution to exercise three of the whispering gophers codelab.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"

	"code.google.com/p/whispering-gophers/util"
)

var master = flag.String("master", "", "master peer address")

type Message struct {
	Body string
	Addr string
}

func main() {
	flag.Parse()
	if *master == "" {
		log.Println("missing -master")
		flag.Usage()
		return
	}

	// Start listening so we don't miss any message
	l, err := util.Listen()
	if err != nil {
		log.Fatal(err)
	}

	// Announce yourself by sending a message to the master
	c, err := net.Dial("tcp", *master)
	if err != nil {
		log.Fatalf("opening connection to master: %v", err)
	}

	m := Message{
		Addr: l.Addr().String(),
	}
	err = json.NewEncoder(c).Encode(m)
	if err != nil {
		log.Fatalf("sending announcement: %v", err)
	}

	// Accept and handle incoming connections
	for {
		c, err := l.Accept()
		if err != nil {
			log.Fatalf("accepting new connection: %v", err)
		}
		go handleConnection(c)
	}
}

func handleConnection(c net.Conn) {
	log.Println("Got new connection", c.RemoteAddr())
	dec := json.NewDecoder(c)
	var msg Message
	for {
		err := dec.Decode(&msg)
		if err != nil {
			log.Println(err)
			break
		}
		fmt.Println(msg.Body)
	}
}
