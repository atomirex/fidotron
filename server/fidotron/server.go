package fidotron

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

type Server struct {
	broker *Broker
}

func NewServer(b *Broker) *Server {
	return &Server{
		broker: b,
	}
}

type streamingConnection struct {
	ws            *websocket.Conn
	s             *Server
	outbox        chan *Message
	subscriptions map[string]bool
}

func newStreamingConnection(ws *websocket.Conn, s *Server) *streamingConnection {
	return &streamingConnection{
		ws:            ws,
		outbox:        make(chan *Message),
		s:             s,
		subscriptions: make(map[string]bool),
	}
}

func (sc *streamingConnection) Write(update *Update) {
	sc.outbox <- &Message{Cmd: CmdUpdate, Topic: update.Topic, Payload: string(update.Payload)}
}

func (sc *streamingConnection) Subscribed(pattern string) {
	if sc.subscriptions != nil {
		sc.subscriptions[pattern] = true
		sc.outbox <- &Message{Cmd: CmdSubscriptionStarted, Topic: pattern}
	}
}

func (sc *streamingConnection) Unsubscribed(pattern string) {
	if sc.subscriptions != nil {
		delete(sc.subscriptions, pattern)
		sc.outbox <- &Message{Cmd: CmdSubscriptionStopped, Topic: pattern}
	}
}

func (sc *streamingConnection) clearSubscriptions() {
	// The nil gating here might prevent chaos
	// In reality this whole thing is ugly
	subs := sc.subscriptions
	sc.subscriptions = nil

	for p := range subs {
		sc.s.broker.Unsubscribe(p, sc)
	}
}

func (sc *streamingConnection) run() {
	defer sc.clearSubscriptions()

	terminating := make(chan bool, 1)

	go func() {
		for {
			msg := &Message{}
			err := sc.ws.ReadJSON(&msg)
			if err != nil {
				terminating <- true
				return
			}

			switch msg.Cmd {
			case CmdSubscriptionRequest:
				sc.s.broker.Subscribe(msg.Topic, sc)
				break
			case CmdUnsubscriptionRequest:
				sc.s.broker.Unsubscribe(msg.Topic, sc)
				break
			case CmdUpdate:
				sc.s.broker.Send(msg.Topic, []byte(msg.Payload))
				break
			}
		}
	}()

	for {
		select {
		case msg := <-sc.outbox:
			err := sc.ws.WriteJSON(msg)
			if err != nil {
				return
			}
			break
		case <-terminating:
			terminating <- true
			return
		}
	}
}

func (s *Server) Start() {
	var upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}

	http.HandleFunc("/websocket", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println(err)
			return
		}

		sc := newStreamingConnection(conn, s)
		sc.run()
	})

	http.HandleFunc("/push", func(rw http.ResponseWriter, r *http.Request) {
		s.broker.Send(r.FormValue("topic"), []byte(r.FormValue("payload")))
		fmt.Fprintf(rw, "OK")
	})

	http.Handle("/", http.FileServer(http.Dir("../static")))
	http.ListenAndServe(":8080", nil)
}
