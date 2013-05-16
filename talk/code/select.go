package main

import "fmt"

func main() {
	// START OMIT
	ch := make(chan int)

	select {
	case ch <- 42:
		fmt.Println("Send succeded")
	default:
		fmt.Println("Send failed")
	}
	// END OMIT
}
