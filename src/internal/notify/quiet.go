// Package notify считает, когда можно отправить уведомление клиенту с учётом тихих часов.
package notify

import (
	"fmt"
	"time"

	"github.com/keepmeinyourmind/zapas/src/internal/domain"
)

// Значения по умолчанию, если клиент не задал свои тихие часы.
const (
	DefaultQuietFrom = "23:00"
	DefaultQuietTo   = "09:00"
	DefaultTimezone  = "Europe/Moscow"
)

// Quiet — тихие часы клиента: с From до To по местному времени (минуты от полуночи).
// Интервал может переходить через полночь: 23:00–09:00.
type Quiet struct {
	From, To int
	Loc      *time.Location
}

// ParseQuiet разбирает тихие часы из правила. Пустые значения — значения по умолчанию.
func ParseQuiet(from, to, tz string) (Quiet, error) {
	if from == "" {
		from = DefaultQuietFrom
	}
	if to == "" {
		to = DefaultQuietTo
	}
	if tz == "" {
		tz = DefaultTimezone
	}
	f, err := parseClock(from)
	if err != nil {
		return Quiet{}, err
	}
	t, err := parseClock(to)
	if err != nil {
		return Quiet{}, err
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return Quiet{}, domain.ValidationError("Неизвестный часовой пояс: " + tz)
	}
	return Quiet{From: f, To: t, Loc: loc}, nil
}

func parseClock(s string) (int, error) {
	t, err := time.Parse("15:04", s)
	if err != nil {
		return 0, domain.ValidationError(fmt.Sprintf("Время должно быть в формате ЧЧ:ММ, получено %q", s))
	}
	return t.Hour()*60 + t.Minute(), nil
}

// NotifyAt — когда отправить уведомление о событии, случившемся в момент t.
// Вне тихих часов — сразу (t). В тихие часы — в момент их окончания (To) по местному времени.
// Результат в UTC, как всё время в системе.
func NotifyAt(t time.Time, q Quiet) time.Time {
	if q.From == q.To { // тихие часы не заданы
		return t.UTC()
	}
	local := t.In(q.Loc)
	minute := local.Hour()*60 + local.Minute()

	var inQuiet bool
	if q.From < q.To { // интервал внутри суток, например 13:00–15:00
		inQuiet = minute >= q.From && minute < q.To
	} else { // через полночь, например 23:00–09:00
		inQuiet = minute >= q.From || minute < q.To
	}
	if !inQuiet {
		return t.UTC()
	}

	end := time.Date(local.Year(), local.Month(), local.Day(), q.To/60, q.To%60, 0, 0, q.Loc)
	if !end.After(local) { // уже после полуночи нужен «завтра»: 23:30 → 09:00 следующего дня
		end = end.AddDate(0, 0, 1)
	}
	return end.UTC()
}
