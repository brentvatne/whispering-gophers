package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
)

var address = flag.String("addr", "", "address to listen")

type dumpWriter struct {
	c net.Conn
	w io.Writer
}

func (w dumpWriter) Write(v []byte) (int, error) {
	fmt.Fprintf(w.w, "[%v->%v]", w.c.RemoteAddr(), w.c.LocalAddr())
	return w.w.Write(v)
}

func main() {
	flag.Parse()
	if *address == "" {
		log.Println("Missing value for -addr")
		flag.Usage()
		return
	}
	l, err := net.Listen("tcp", *address)
	if err != nil {
		log.Fatal(err)
	}
	for {
		c, err := l.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		go io.Copy(dumpWriter{c, os.Stdout}, c)
	}
}
