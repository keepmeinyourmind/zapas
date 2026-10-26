package domain

import "slices"

// Median возвращает медиану сумм: середину после сортировки.
// Для чётного количества — среднее двух серединных, для пустого списка — 0.
// Исходный слайс не меняет.
func Median(amounts []Money) Money {
	if len(amounts) == 0 {
		return 0
	}
	s := slices.Clone(amounts)
	slices.Sort(s)
	mid := len(s) / 2
	if len(s)%2 == 1 {
		return s[mid]
	}
	return (s[mid-1] + s[mid]) / 2
}
