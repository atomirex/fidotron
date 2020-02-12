package fidotron

import "testing"

func testPatternRaw(t *testing.T, pattern string, raw string) {
	p := NewPattern(pattern)

	if p.Raw != raw {
		t.Errorf("RAW match failure %s != %s", p.Raw, raw)
	}
}
func TestPatternSetup(t *testing.T) {
	testPatternRaw(t, "yes", "yes")
	testPatternRaw(t, "yes/", "yes")
	testPatternRaw(t, "/yes", "yes")
	testPatternRaw(t, "/yes/", "yes")
	testPatternRaw(t, "hello/world", "hello/world")
	testPatternRaw(t, "/hello/world", "hello/world")
	testPatternRaw(t, "hello/world/", "hello/world")
	testPatternRaw(t, "/hello/world/", "hello/world")
}

func testPatternMatching(t *testing.T, pattern string, query string, shouldmatch bool) {
	p := NewPattern(pattern)
	matches := p.Match(query)
	if shouldmatch != matches {
		if matches {
			t.Errorf("Erroneous match occured between pattern %s and query %s", pattern, query)
		} else {
			t.Errorf("Erroneous failure to match occured between pattern %s and query %s", pattern, query)
		}
	}
}
func TestPatternMatch(t *testing.T) {
	testPatternMatching(t, "+", "yes", true)
	testPatternMatching(t, "+/yes", "yes", false)
	testPatternMatching(t, "#", "yes", true)
	testPatternMatching(t, "yes", "yes", true)
	testPatternMatching(t, "yes", "no", false)
	testPatternMatching(t, "hello/world", "hello/world", true)
	testPatternMatching(t, "+/world", "hello/world", true)
	testPatternMatching(t, "+/+", "hello/world", true)
	testPatternMatching(t, "hello/+", "hello/world", true)
	testPatternMatching(t, "hello/#", "hello/world", true)
	testPatternMatching(t, "hello/+/whatever", "hello/world", false)
	testPatternMatching(t, "hello/+/whatever", "hello/world/whatever", true)
	testPatternMatching(t, "hello/#", "hello/world/whatever", true)
	testPatternMatching(t, "hello/+", "hello/world/whatever", false)
}

type testsub struct {
}

func (ts *testsub) Subscribed(pattern string) {

}

func (ts *testsub) Unsubscribed(pattern string) {

}

func (ts *testsub) Write(update *Update) {

}

func testPatternMatcher(t *testing.T, pattern string, query string, shouldmatch bool) {
	ts := &testsub{}
	m := NewMatcher()
	m.AddSubscription(&Subscription{Pattern: NewPattern(pattern), Subscriber: ts})
	r, _ := m.Match(query)

	matches := len(r) > 0
	if shouldmatch != matches {
		if matches {
			t.Errorf("Erroneous match occured between pattern %s and query %s", pattern, query)
		} else {
			t.Errorf("Erroneous failure to match occured between pattern %s and query %s", pattern, query)
		}
	}
}
func TestMatcher(t *testing.T) {
	testPatternMatcher(t, "+", "yes", true)
	testPatternMatcher(t, "+/yes", "yes", false)
	testPatternMatcher(t, "#", "yes", true)
	testPatternMatcher(t, "yes", "yes", true)
	testPatternMatcher(t, "yes", "no", false)
	testPatternMatcher(t, "hello/world", "hello/world", true)
	testPatternMatcher(t, "+/world", "hello/world", true)
	testPatternMatcher(t, "+/+", "hello/world", true)
	testPatternMatcher(t, "hello/+", "hello/world", true)
	testPatternMatcher(t, "hello/#", "hello/world", true)
	testPatternMatcher(t, "hello/+/whatever", "hello/world", false)
	testPatternMatcher(t, "hello/+/whatever", "hello/world/whatever", true)
	testPatternMatcher(t, "hello/#", "hello/world/whatever", true)
	testPatternMatcher(t, "hello/world/#", "hello/world/whatever", true)
	testPatternMatcher(t, "hello/+", "hello/world/whatever", false)
	testPatternMatcher(t, "hello/world/+", "hello/world/whatever", true)
}

func testPatternMatcherBindings(t *testing.T, pattern string, query string, bindings map[string]string) {
	ts := &testsub{}
	m := NewMatcher()
	m.AddSubscription(&Subscription{Pattern: NewPattern(pattern), Subscriber: ts})
	_, r := m.Match(query)

	for _, bound := range r {
		for k, v := range bindings {
			b, ok := bound[k]
			if !ok {
				t.Errorf("Failed to attach binding for identifier %s in pattern %s and query %s", k, pattern, query)
			} else {
				if b != v {
					t.Errorf("Wrong value %s expecting %s on binding for identifier %s in pattern %s and query %s", b, v, k, pattern, query)
				}
			}
		}

		for k, v := range bound {
			_, ok := bindings[k]
			if !ok {
				t.Errorf("Unexpected binding %s:%s in pattern %s and query %s", k, v, pattern, query)
			}
		}
	}
}
func TestMatcherBindings(t *testing.T) {
	bindings := make(map[string]string)
	bindings["name"] = "value"

	empty := make(map[string]string)

	dualbindings := make(map[string]string)
	dualbindings["first"] = "1st"
	dualbindings["second"] = "2nd"

	longremainer := make(map[string]string)
	longremainer["this"] = "world/whatever"

	testPatternMatcherBindings(t, "+name", "value", bindings)
	testPatternMatcherBindings(t, "+/yes", "yes", empty)
	testPatternMatcherBindings(t, "#name", "value", bindings)
	testPatternMatcherBindings(t, "yes", "yes", empty)
	testPatternMatcherBindings(t, "yes", "no", empty)
	testPatternMatcherBindings(t, "hello/world", "hello/world", empty)
	testPatternMatcherBindings(t, "+name/world", "value/world", bindings)
	testPatternMatcherBindings(t, "+first/+second", "1st/2nd", dualbindings)
	testPatternMatcherBindings(t, "hello/+name", "hello/value", bindings)
	testPatternMatcherBindings(t, "hello/#name", "hello/value", bindings)
	testPatternMatcherBindings(t, "hello/+/whatever", "hello/world", empty)
	testPatternMatcherBindings(t, "hello/+name/whatever", "hello/value/whatever", bindings)
	testPatternMatcherBindings(t, "hello/#this", "hello/world/whatever", longremainer)
	testPatternMatcherBindings(t, "hello/world/#name", "hello/world/value", bindings)
	testPatternMatcherBindings(t, "hello/+", "hello/world/whatever", empty)
	testPatternMatcherBindings(t, "hello/world/+name", "hello/world/value", bindings)
}
