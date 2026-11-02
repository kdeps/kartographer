package text

import "testing"

func TestTopic(t *testing.T) {
	var tp Topic
	if !tp.Empty() || !tp.Related([]string{"billing", "invoice", "stripe"}, 3, 0.2) {
		t.Fatal("an empty thread takes any request")
	}
	tp.Add([]string{"billing", "invoice", "stripe"})
	if tp.Empty() {
		t.Fatal("added words are kept")
	}
	if !tp.Related([]string{"yes"}, 3, 0.2) {
		t.Error("a short request is a follow-up")
	}
	if !tp.Related([]string{"billing", "monthly", "cycle", "date", "proration"}, 3, 0.2) {
		t.Error("one shared word in five meets a 0.2 share")
	}
	if tp.Related([]string{"kubernetes", "ingress", "certificate", "renewal"}, 3, 0.2) {
		t.Error("no shared words starts a new topic")
	}
}
