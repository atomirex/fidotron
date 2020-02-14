package fidotron

import "fmt"

const (
	CmdUpdate = iota
	CmdError
	CmdSubscriptionRequest
	CmdSubscriptionStarted
	CmdUnsubscriptionRequest
	CmdSubscriptionStopped
)

type WSMessage struct {
	Cmd     int
	Topic   string
	Payload string
}

func (wm *WSMessage) String() string {
	return fmt.Sprintf("Cmd: %d Topic: %s Payload %s", wm.Cmd, wm.Topic, wm.Payload)
}

type Update struct {
	Topic   string
	Payload []byte
}

type Subscriber interface {
	Subscribed(pattern string)
	Unsubscribed(pattern string)
	Write(update *Update)
}

type simpleSubscriber struct {
	f func(topic string, payload []byte)
}

func BasicSubscriber(f func(topic string, payload []byte)) *simpleSubscriber {
	return &simpleSubscriber{f: f}
}

func (ss *simpleSubscriber) Subscribed(pattern string) {

}

func (ss *simpleSubscriber) Unsubscribed(pattern string) {

}

func (ss *simpleSubscriber) Write(update *Update) {
	ss.f(update.Topic, update.Payload)
}

type Pattern struct {
	Raw      string
	Sections []string
}

func (p *Pattern) String() string {
	return p.Raw
}

type Subscription struct {
	Pattern    *Pattern
	Subscriber Subscriber
}
