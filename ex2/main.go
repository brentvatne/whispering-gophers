// Solution to exercise two of the whispering gophers codelab.
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"log"
	"net"
	"os"
)

var server = flag.String("server", "", "server address")

type Message struct {
	Body string
}

func main() {
	flag.Parse()
	if *server == "" {
		log.Println("Please specify a server address to connect")
		flag.Usage()
		return
	}

	r := bufio.NewReader(os.Stdin)
	c, err := net.Dial("tcp", *server)
	if err != nil {
		log.Fatal(err)
	}
	w := json.NewEncoder(c)

	for {
		line, err := r.ReadString('\n')
		if l := len(line); l > 1 {
			err := w.Encode(Message{line[:l-1]})
			if err != nil {
				log.Fatal(err)
			}
		}
		if err != nil {
			log.Fatal(err)
		}
	}
}
