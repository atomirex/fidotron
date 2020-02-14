package fidotron

import (
	"log"
	"strings"
)

type Broker struct {
	matcher              *Matcher
	inbox                chan *Update
	subscriptionRequests chan *Subscription
	removeRequests       chan *Subscription
}

func stringToTokens(p string) []string {
	s := strings.Split(p, "/")
	t := make([]string, 0)
	for i := 0; i < len(s); i++ {
		if s[i] != "" {
			t = append(t, s[i])
		}
	}
	return t
}

func NewPattern(p string) *Pattern {
	t := stringToTokens(p)
	pattern := &Pattern{Raw: strings.Join(t, "/"), Sections: t}
	return pattern
}

func (p *Pattern) Match(other string) bool {
	t := stringToTokens(other)

	if len(p.Sections) > len(t) {
		return false
	}

	for i := 0; i < len(p.Sections); i++ {
		switch p.Sections[i] {
		case "#":
			return true
		case "+":
			if i >= (len(p.Sections) - 1) {
				return len(p.Sections) == len(t)
			}
			break
		default:
			if p.Sections[i] != t[i] {
				return false
			}
			break
		}
	}
	return true
}

type Matcher struct {
	root *matchNode
}

type matchNode struct {
	children    map[string]*matchNode
	subscribers map[Subscriber]bool
	wildcards   map[string]map[Subscriber]bool
	remainers   map[string]map[Subscriber]bool
}

func (n *matchNode) match(output map[Subscriber]bool, bindings map[Subscriber]map[string]string, path []string, index int) {
	log.Println("Digging index", index)
	if index < len(path) {
		if n.children["#"] != nil {
			for s := range n.children["#"].subscribers {
				output[s] = true
			}
		}

		for id, subs := range n.remainers {
			remaining := strings.Join(path[index:], "/")
			for s := range subs {
				if bindings[s] == nil {
					bindings[s] = make(map[string]string)
				}
				bindings[s][id] = remaining
			}
		}

		if n.children["+"] != nil {
			n.children["+"].match(output, bindings, path, index+1)
		}

		for id, subs := range n.wildcards {
			for s := range subs {
				log.Println("Looking at wildcard", id, "for candidate", path[index])
				if bindings[s] == nil {
					bindings[s] = make(map[string]string)
				}
				bindings[s][id] = path[index]
				log.Println("Values bound", s, id, path[index], index)
			}
		}

		if n.children[path[index]] != nil {
			n.children[path[index]].match(output, bindings, path, index+1)
		}
	} else if index == len(path) {
		for s := range n.subscribers {
			output[s] = true
		}
	}
}

func (n *matchNode) addSubscription(sub Subscriber, path []string, index int) {
	if index < len(path) {
		if path[index][0] == '#' {
			p := path[index][1:]
			if len(p) > 0 {
				if n.remainers[p] == nil {
					n.remainers[p] = make(map[Subscriber]bool)
				}
				n.remainers[p][sub] = true
			}
		}

		if path[index][0] == '+' {
			p := path[index][1:]
			if len(p) > 0 {
				if n.wildcards[p] == nil {
					n.wildcards[p] = make(map[Subscriber]bool)
				}
				n.wildcards[p][sub] = true
			}
		}

		if n.children[path[index]] == nil {
			n.children[path[index]] = newMatchNode()
		}

		n.children[path[index]].addSubscription(sub, path, index+1)
	} else if index == len(path) {
		n.subscribers[sub] = true
	}
}

func (n *matchNode) removeSubscription(sub Subscriber, path []string, index int) {
	// TODO wildcards
	// TODO remainers
	// FIXME looks like this leaks dangling garbage of matchNodes with no subs?
	if index < len(path) {
		if n.children[path[index]] == nil {
			return
		}

		n.children[path[index]].removeSubscription(sub, path, index+1)
	} else if index == len(path) {
		delete(n.subscribers, sub)
	}
}

func newMatchNode() *matchNode {
	return &matchNode{
		children:    make(map[string]*matchNode),
		subscribers: make(map[Subscriber]bool),
		wildcards:   make(map[string]map[Subscriber]bool),
		remainers:   make(map[string]map[Subscriber]bool),
	}
}

func NewMatcher() *Matcher {
	return &Matcher{root: newMatchNode()}
}

func (m *Matcher) AddSubscription(sub *Subscription) {
	m.root.addSubscription(sub.Subscriber, sub.Pattern.Sections, 0)
}

func (m *Matcher) RemoveSubscription(sub *Subscription) {
	m.root.removeSubscription(sub.Subscriber, sub.Pattern.Sections, 0)
}

func (m *Matcher) Match(topic string) (map[Subscriber]bool, map[Subscriber]map[string]string) {
	path := stringToTokens(topic)

	out := make(map[Subscriber]bool)
	bind := make(map[Subscriber]map[string]string)
	m.root.match(out, bind, path, 0)

	return out, bind
}

func NewBroker() *Broker {
	b := &Broker{
		matcher:              NewMatcher(),
		inbox:                make(chan *Update),
		subscriptionRequests: make(chan *Subscription),
		removeRequests:       make(chan *Subscription),
	}

	go func() {
		for {
			select {
			case msg := <-b.inbox:
				subs, _ := b.matcher.Match(msg.Topic)

				for l := range subs {
					l.Write(msg)
				}
				break
			case s := <-b.subscriptionRequests:
				b.matcher.AddSubscription(s)
				s.Subscriber.Subscribed(s.Pattern.Raw)
				break
			case s := <-b.removeRequests:
				b.matcher.RemoveSubscription(s)
				s.Subscriber.Unsubscribed(s.Pattern.Raw)
				break
			}
		}
	}()

	return b
}

func (b *Broker) Send(topic string, p []byte) {
	// Copy the message before we return as some senders like reusing the same buffer
	c := make([]byte, len(p))
	copy(c, p)

	u := &Update{
		Topic:   topic,
		Payload: c,
	}

	b.inbox <- u
}

func (b *Broker) Subscribe(pattern string, l Subscriber) {
	b.subscriptionRequests <- &Subscription{Pattern: NewPattern(pattern), Subscriber: l}
}

func (b *Broker) Unsubscribe(pattern string, l Subscriber) {
	b.removeRequests <- &Subscription{Pattern: NewPattern(pattern), Subscriber: l}
}
