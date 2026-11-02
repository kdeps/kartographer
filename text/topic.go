package text

// Topic is the words of one conversation thread, to tell a request that
// continues it from one that starts something new. The caller turns text into
// words and sets the thresholds, so Topic holds no policy of its own.
type Topic struct {
	words map[string]bool
}

// Empty reports whether the thread has no words yet.
func (t *Topic) Empty() bool { return len(t.words) == 0 }

// Related reports whether a request with these content words continues the
// thread: any request continues an empty thread, a request with fewer than
// followUpWords words is a follow-up, and otherwise at least relatedShare of
// its words must already be in the thread.
func (t *Topic) Related(ws []string, followUpWords int, relatedShare float64) bool {
	if t.Empty() || len(ws) < followUpWords {
		return true
	}
	shared := 0
	for _, w := range ws {
		if t.words[w] {
			shared++
		}
	}
	return float64(shared)/float64(len(ws)) >= relatedShare
}

// Add adds words to the thread.
func (t *Topic) Add(ws []string) {
	if t.words == nil {
		t.words = map[string]bool{}
	}
	for _, w := range ws {
		t.words[w] = true
	}
}
