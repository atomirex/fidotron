package fidotron

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"golang.org/x/net/websocket"
)

type Server struct {
	broker     *Broker
	appManager *AppManager
	runner     *Runner
	uploads    *uploadedCache
}

func NewServer(b *Broker, am *AppManager, r *Runner) *Server {
	return &Server{
		broker:     b,
		appManager: am,
		runner:     r,
		uploads:    newUploadedCache(),
	}
}

type streamingConnection struct {
	ws            *websocket.Conn
	s             *Server
	outbox        chan *WSMessage
	subscriptions map[string]bool
}

func newStreamingConnection(ws *websocket.Conn, s *Server) *streamingConnection {
	return &streamingConnection{
		ws:            ws,
		outbox:        make(chan *WSMessage),
		s:             s,
		subscriptions: make(map[string]bool),
	}
}

func (sc *streamingConnection) Write(update *Update) {
	sc.outbox <- &WSMessage{Cmd: CmdUpdate, Topic: update.Topic, Payload: string(update.Payload)}
}

func (sc *streamingConnection) Subscribed(pattern string) {
	if sc.subscriptions != nil {
		sc.subscriptions[pattern] = true
		sc.outbox <- &WSMessage{Cmd: CmdSubscriptionStarted, Topic: pattern}
	}
}

func (sc *streamingConnection) Unsubscribed(pattern string) {
	if sc.subscriptions != nil {
		delete(sc.subscriptions, pattern)
		sc.outbox <- &WSMessage{Cmd: CmdSubscriptionStopped, Topic: pattern}
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
			msg := &WSMessage{}
			sc.ws.SetReadDeadline(time.Now().Add(4 * time.Second))
			err := websocket.JSON.Receive(sc.ws, &msg)
			if err != nil {
				fmt.Println("Receive error " + err.Error())
				terminating <- true
				return
			}

			switch msg.Cmd {
			case CmdPing:
				sc.outbox <- &WSMessage{Cmd: CmdPong}
				break
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
			sc.ws.SetWriteDeadline(time.Now().Add(4 * time.Second))
			b, _ := json.Marshal(msg)
			_, err := sc.ws.Write(b)
			if err != nil {
				fmt.Println("Send error " + err.Error())
				// TODO terminate the receiver?
				return
			}
			break
		case <-terminating:
			terminating <- true
			return
		}
	}
}

type uploadedCache struct {
	uploads *sync.Map
}

func newUploadedCache() *uploadedCache {
	u := &uploadedCache{
		uploads: &sync.Map{},
	}

	return u
}

func (u *uploadedCache) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s := r.URL.Path[len("/uploaded/"):]
	b, ok := u.uploads.Load(s)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	http.ServeContent(w, r, s, time.Now(), bytes.NewReader(b.([]byte)))
}

func (s *Server) Start() {
	http.Handle("/websocket", websocket.Handler(func(ws *websocket.Conn) {
		sc := newStreamingConnection(ws, s)
		sc.run()
	}))

	http.HandleFunc("/push", func(rw http.ResponseWriter, r *http.Request) {
		s.broker.Send(r.FormValue("topic"), []byte(r.FormValue("payload")))
		fmt.Fprintf(rw, "OK")
	})

	http.HandleFunc("/runapp/", func(rw http.ResponseWriter, r *http.Request) {
		app := r.URL.Path[len("/runapp/"):]
		a := s.appManager.App(app)
		// TODO should probably be PIDs not names
		s.runner.Run(a, a.Name+"/stdout", a.Name+"/stderr")
		fmt.Fprintf(rw, "OK")
	})

	http.HandleFunc("/upload", func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseMultipartForm(2000000000)
		if err != nil {
			fmt.Println("Error parsing form " + err.Error())
			return
		}

		var b bytes.Buffer

		file, header, err := r.FormFile("file")
		if file != nil {
			defer file.Close()
		}

		if err != nil {
			fmt.Println("ERROR " + err.Error())
			return
		}

		io.Copy(&b, file)

		s.uploads.uploads.Store(header.Filename, b.Bytes())

		b.Reset()

		http.Redirect(w, r, "/uploaded/"+header.Filename, http.StatusSeeOther)

		// TODO publish that a file was upload to the relevant topic!
		return
	})

	http.Handle("/uploaded/", s.uploads)

	http.Handle("/", http.FileServer(http.Dir("../static")))
	http.ListenAndServe(":8080", nil)
}
