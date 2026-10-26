package notify

import (
	"testing"
	"time"
)

func TestNotifyAt(t *testing.T) {
	msk, _ := time.LoadLocation("Europe/Moscow")
	at := func(d, h, m int) time.Time { return time.Date(2026, 10, d, h, m, 0, 0, msk) }

	night, err := ParseQuiet("23:00", "09:00", "Europe/Moscow")
	if err != nil {
		t.Fatal(err)
	}
	day, err := ParseQuiet("13:00", "15:00", "Europe/Moscow")
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		q    Quiet
		in   time.Time
		want time.Time
	}{
		{"днём — сразу", night, at(25, 14, 0), at(25, 14, 0)},
		{"ровно в 23:00 — уже тихо, до 09:00 завтра", night, at(25, 23, 0), at(26, 9, 0)},
		{"в 23:30 — до 09:00 завтра", night, at(25, 23, 30), at(26, 9, 0)},
		{"в 03:00 — до 09:00 сегодня", night, at(26, 3, 0), at(26, 9, 0)},
		{"ровно в 09:00 — уже можно", night, at(26, 9, 0), at(26, 9, 0)},
		{"в 22:59 — ещё можно", night, at(25, 22, 59), at(25, 22, 59)},
		{"интервал внутри суток: 14:00 → 15:00", day, at(25, 14, 0), at(25, 15, 0)},
		{"интервал внутри суток: 16:00 — сразу", day, at(25, 16, 0), at(25, 16, 0)},
		{"тихие часы выключены (from == to) — сразу", Quiet{From: 0, To: 0, Loc: msk}, at(26, 3, 0), at(26, 3, 0)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NotifyAt(tt.in, tt.q)
			if !got.Equal(tt.want) {
				t.Errorf("NotifyAt(%s) = %s, want %s", tt.in.Format("02.01 15:04"),
					got.In(msk).Format("02.01 15:04"), tt.want.Format("02.01 15:04"))
			}
			if got.Location() != time.UTC {
				t.Errorf("результат должен быть в UTC, получено %s", got.Location())
			}
		})
	}
}

// Доход пришёл ночью по UTC — важно считать по местному времени клиента.
func TestNotifyAtUsesClientTimezone(t *testing.T) {
	q, _ := ParseQuiet("23:00", "09:00", "Asia/Vladivostok") // UTC+10
	in := time.Date(2026, 10, 25, 14, 0, 0, 0, time.UTC)     // 00:00 во Владивостоке
	want := time.Date(2026, 10, 25, 23, 0, 0, 0, time.UTC)   // 09:00 во Владивостоке
	if got := NotifyAt(in, q); !got.Equal(want) {
		t.Errorf("NotifyAt = %s, want %s", got, want)
	}
}

func TestParseQuiet(t *testing.T) {
	if _, err := ParseQuiet("", "", ""); err != nil {
		t.Errorf("пустые значения должны давать значения по умолчанию, а не ошибку: %v", err)
	}
	for _, bad := range [][3]string{{"25:00", "09:00", ""}, {"23:00", "9", ""}, {"23:00", "09:00", "Mars/Olympus"}} {
		if _, err := ParseQuiet(bad[0], bad[1], bad[2]); err == nil {
			t.Errorf("ParseQuiet(%q) должна вернуть ошибку", bad)
		}
	}
}
