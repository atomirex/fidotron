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

	am := fidotron.NewAppManager()

	r := fidotron.NewRunner(b)

	s := fidotron.NewServer(b, am, r)
	s.Start()
}
