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

	am.Add(&fidotron.App{
		Name: "littleapp",
		Args: nil,
		Dir:  "",
		Path: "bin/littleapp",
	})

	am.Add(&fidotron.App{
		Name: "mdns",
		Args: nil,
		Dir:  "",
		Path: "bin/mdns",
	})

	am.Add(&fidotron.App{
		Name: "nanoleaf",
		Args: nil,
		Dir:  "",
		Path: "bin/nanoleaf",
	})

	r := fidotron.NewRunner(b)

	s := fidotron.NewServer(b, am, r)
	s.Start()
}
