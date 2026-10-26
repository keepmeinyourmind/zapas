package domain

import (
	"slices"
	"testing"
)

func TestMedian(t *testing.T) {
	tests := []struct {
		name string
		in   []Money
		want Money
	}{
		{"пустой список", nil, 0},
		{"одна сумма", []Money{FromRubles(30000)}, FromRubles(30000)},
		{"нечётное количество, не по порядку", []Money{FromRubles(36000), FromRubles(24000), FromRubles(30000)}, FromRubles(30000)},
		{"чётное количество — среднее двух серединных", []Money{FromRubles(20000), FromRubles(10000)}, FromRubles(15000)},
		{"выброс не сдвигает медиану", []Money{FromRubles(30000), FromRubles(30000), FromRubles(90000)}, FromRubles(30000)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Median(tt.in); got != tt.want {
				t.Errorf("Median(%v) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

// Медиана не должна переставлять исходные суммы: слайс — это ссылка на общий массив,
// и сортировка «на месте» испортила бы историю у того, кто вызвал функцию.
func TestMedianDoesNotModifyInput(t *testing.T) {
	in := []Money{FromRubles(36000), FromRubles(24000), FromRubles(30000)}
	orig := slices.Clone(in)
	Median(in)
	if !slices.Equal(in, orig) {
		t.Errorf("Median изменила входной слайс: было %v, стало %v", orig, in)
	}
}
