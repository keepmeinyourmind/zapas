// Package clock — текущее время сервиса. В обычной работе это настоящее время,
// в демо его можно зафиксировать (сценарии) и сдвигать вперёд (таймауты вопросов).
package clock

import (
	"sync"
	"time"
)

// Clock безопасен для одновременного использования из HTTP-обработчиков и фонового таймера:
// mutex не даёт двум горутинам менять время одновременно.
type Clock struct {
	mu    sync.Mutex
	fixed *time.Time    // nil — настоящее время
	shift time.Duration // насколько сдвинуто вперёд
}

func New() *Clock { return &Clock{} }

// Now — текущее время в UTC.
func (c *Clock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	base := time.Now()
	if c.fixed != nil {
		base = *c.fixed
	}
	return base.Add(c.shift).UTC()
}

// Set фиксирует время (для сценариев демо) и сбрасывает сдвиг.
func (c *Clock) Set(t time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.fixed, c.shift = &t, 0
}

// Advance сдвигает время вперёд (для демо: «прошло 25 часов»).
func (c *Clock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.shift += d
}
