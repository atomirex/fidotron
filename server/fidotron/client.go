package fidotron

import (
	"fmt"
	"log"
	"time"

	"golang.org/x/net/websocket"
)

/*
TODO

- subscriber patterns with named parameters
- bus to struct adapters
- zero type update batching (to enable structs)
*/

type Client struct {
	outbox    chan interface{}
	broker    *Broker
	subReqs   chan *Subscription
	unsubReqs chan *Subscription
}

func (c *Client) Send(topic string, payload string) {
	c.outbox <- &WSMessage{Cmd: CmdUpdate, Topic: topic, Payload: payload}
}

func (c *Client) Subscribe(pattern string, sub Subscriber) {
	c.subReqs <- &Subscription{Pattern: NewPattern(pattern), Subscriber: sub}
}

func (c *Client) Unsubscribe(pattern string, sub Subscriber) {
	c.unsubReqs <- &Subscription{Pattern: NewPattern(pattern), Subscriber: sub}
}

func NewClient() *Client {
	c := &Client{
		outbox:    make(chan interface{}),
		broker:    NewBroker(),
		subReqs:   make(chan *Subscription),
		unsubReqs: make(chan *Subscription),
	}

	subs := make(map[string]map[*Subscription]bool)

	go func() {
		for {
			ws, err := websocket.Dial("ws://127.0.0.1:8080/websocket", "", "http://127.0.0.1:8080/")
			if err != nil {
				log.Fatal(err)
			}

			go func() {
				for {
					select {
					case s := <-c.subReqs:
						c.broker.Subscribe(s.Pattern.String(), s.Subscriber)
						_, ok := subs[s.Pattern.String()]
						if !ok {
							subs[s.Pattern.String()] = make(map[*Subscription]bool)
						}
						subs[s.Pattern.String()][s] = true
						break
					case s := <-c.unsubReqs:
						_, ok := subs[s.Pattern.String()]
						if ok {
							delete(subs[s.Pattern.String()], s)
							if len(subs[s.Pattern.String()]) == 0 {
								delete(subs, s.Pattern.String())
							}

							c.broker.Unsubscribe(s.Pattern.String(), s.Subscriber)
						}
						break
					case msg := <-c.outbox:
						err = websocket.JSON.Send(ws, msg)
						if err != nil {
							log.Fatal(err)
						}
						break
					case <-time.After(2 * time.Second):
						err = websocket.JSON.Send(ws, &WSMessage{Cmd: CmdPing})
						if err != nil {
							log.Fatal(err)
						}
						break
					}
				}
			}()

			for {
				msg := &WSMessage{}
				ws.SetReadDeadline(time.Now().Add(4 * time.Second))
				err = websocket.JSON.Receive(ws, &msg)
				if err != nil {
					log.Fatal(err)
				}

				switch msg.Cmd {
				case CmdUpdate:
					c.broker.Send(msg.Topic, []byte(msg.Payload))
				case CmdPing:
					c.outbox <- &WSMessage{Cmd: CmdPong}
					break
				case CmdPong:
					break
				case CmdError:
					log.Fatal("Error received from server")
				}
			}
		}

		fmt.Println("Ending goroutine for unknown reason")
	}()

	c.outbox <- &WSMessage{Cmd: CmdPing}
	c.outbox <- &WSMessage{Cmd: CmdSubscriptionRequest, Topic: "#"}

	return c
}
