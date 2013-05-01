// Solution to exercise one of the whispering gophers codelab.
package main

import (
	"bufio"
	"encoding/json"
	"log"
	"os"
)

type Message struct{ Body string }

func main() {
	r := bufio.NewReader(os.Stdin)
	w := json.NewEncoder(os.Stdout)
	for {
		// Read one line.
		line, err := r.ReadString('\n')
		// If there's any data, encode it into a message.
		if l := len(line); l > 1 {
			err := w.Encode(Message{line[:l-1]})
			if err != nil {
				log.Println(err)
				continue
			}
		}
		// If there's an error, print it and stop.
		if err != nil {
			log.Println(err)
			break
		}
	}
}
