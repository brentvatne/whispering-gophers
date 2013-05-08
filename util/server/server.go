package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"

	"code.google.com/p/whispering-gophers/util"
)

type dumpWriter struct {
	c net.Conn
	w io.Writer
}

func (w dumpWriter) Write(v []byte) (int, error) {
	fmt.Fprintf(w.w, "[%v->%v] ", w.c.RemoteAddr(), w.c.LocalAddr())
	return w.w.Write(v)
}

func main() {
	l, err := util.Listen()
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Listening on", l.Addr())
	for {
		c, err := l.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		go io.Copy(dumpWriter{c, os.Stdout}, c)
	}
}
