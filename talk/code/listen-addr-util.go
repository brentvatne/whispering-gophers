package main

import (
	"log"

	"code.google.com/a/whisper/util"
)

func main() {
	l, err := util.Listen()
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Listening on", l.Addr())
}
