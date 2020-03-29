package main

import (
	"encoding/json"
	"fidotron"
	"fmt"
	"log"

	"github.com/atomirex/mdns"
)

func main() {
	fidotron.Run(&mdnsApp{})
}

type mdnsApp struct {
}

func (m *mdnsApp) Prepare() {
	fmt.Println("MDNS prepare")
}

func (m *mdnsApp) Start() {
	fmt.Println("MDNS start")

	push := fidotron.NewClient()

	service := "_services._dns-sd._udp"

	// Make a channel for results and start listening
	entries := make(chan *mdns.ServiceEntry, 4)
	go func() {
		for entry := range entries {
			fmt.Printf("Got new entry: %v\n", entry)
			b, err := json.Marshal(entry)
			if err != nil {
				fmt.Println("Error formatting js for bus")
			} else {
				push.Send("sys/mdns/"+entry.Name, string(b))
			}
		}
	}()

	// Start the lookup
	params := mdns.DefaultParams(service)
	params.Entries = entries

	// Create a new client
	client, err := mdns.NewClient()
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	// Set the multicast interface
	if params.Interface != nil {
		if err := client.SetInterface(params.Interface); err != nil {
			log.Fatal(err)
		}
	}

	// Ensure defaults are set
	if params.Domain == "" {
		params.Domain = "local"
	}

	// Run the query
	client.Query(params)

	close(entries)
}
