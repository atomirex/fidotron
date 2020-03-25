package main

import (
	"fidotron"
	"fmt"
)

const logo = `
  ___
/|'.'|\
 | t |_ 
 FIDOTRON
`

func main() {
	fmt.Println(logo)

	b := fidotron.NewBroker()

	s := fidotron.NewServer(b)
	s.Start()
}
