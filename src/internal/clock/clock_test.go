package clock

import (
	"testing"
	"time"
)

func TestClock(t *testing.T) {
	c := New()
	if d := time.Since(c.Now()); d < 0 || d > time.Second {
		t.Errorf("без фиксации Now должно быть настоящим временем, разница %s", d)
	}
	at := time.Date(2026, 10, 25, 10, 0, 0, 0, time.UTC)
	c.Set(at)
	if !c.Now().Equal(at) {
		t.Errorf("после Set Now = %s, want %s", c.Now(), at)
	}
	c.Advance(25 * time.Hour)
	if want := at.Add(25 * time.Hour); !c.Now().Equal(want) {
		t.Errorf("после Advance Now = %s, want %s", c.Now(), want)
	}
	c.Set(at)
	if !c.Now().Equal(at) {
		t.Error("Set должен сбрасывать сдвиг")
	}
}
