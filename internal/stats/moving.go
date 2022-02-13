/*
yatgo: Yet Another Trader in Go
Copyright (C) 2022  Tim Möhlmann

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published
by the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

// Package stats provides static calculation building blocks for
// trading algoritms.
package stats

// movingList of values, not save for concurrent use.
type movingList[T any] struct {
	entries []T
	pos     int
}

func newMovingList[T any](entries []T) movingList[T] {
	return movingList[T]{entries: entries}
}

// move replaces the oldest value in the list.
func (l *movingList[T]) move(v T) {
	if len(l.entries) == 0 {
		return
	}

	l.entries[l.pos] = v

	if l.pos++; l.pos >= len(l.entries) {
		l.pos = 0
	}
}

type MovingAverage struct {
	list movingList[float64]
}

// Move the list of values by one position.
// Removes the oldest and replaces it by the passed value.
func (ma *MovingAverage) Move(value float64) {
	ma.list.move(value)
}

func (ma MovingAverage) sum() (sum float64) {
	for _, v := range ma.list.entries {
		sum += v
	}

	return sum
}

// Avg returns the current average of the MovingAverage slice.
func (ma MovingAverage) Avg() float64 {
	return ma.sum() / float64(len(ma.list.entries))
}

// AvgIncl calculates the current average with the addional value,
// which can be weighed for partial blocks.
// Weight 1.0 will consider this value with the same weight as all values.
// A lower weight will influence the resulting average less.
func (ma MovingAverage) AvgIncl(value, weight float64) float64 {
	return (value*weight + ma.sum()) / (float64(len(ma.list.entries)) + weight)
}
