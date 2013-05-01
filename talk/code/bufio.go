package main

import (
	"bufio"
	"fmt"
	"log"
	"strings"
)

const input = `A haiku is more
Than just a collection of
Well-formed syllables
`

func main() {
	r := bufio.NewReader(strings.NewReader(input))
	for i := 0; ; i++ {
		s, err := r.ReadString('\n')
		if err != nil {
			log.Println(err)
			break
		}
		fmt.Printf("%v: %q\n", i, s)
	}
}
