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

package stats

import (
	"fmt"
	"math"
	"reflect"
	"strconv"
	"testing"
)

func Test_movingList_move(t *testing.T) {
	tests := []struct {
		name string
		list movingList[int]
		args []int
		want movingList[int]
	}{
		{
			"emtpy",
			movingList[int]{},
			[]int{1, 2},
			movingList[int]{},
		},
		{
			"one",
			newMovingList([]int{1}),
			[]int{2, 3},
			movingList[int]{
				entries: []int{3},
				pos:     0,
			},
		},
		{
			"move one",
			newMovingList([]int{1, 2, 3}),
			[]int{4},
			movingList[int]{
				entries: []int{4, 2, 3},
				pos:     1,
			},
		},
		{
			"move more",
			newMovingList([]int{1, 2, 3}),
			[]int{4, 5, 6, 7},
			movingList[int]{
				entries: []int{7, 5, 6},
				pos:     1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, a := range tt.args {
				tt.list.move(a)
			}

			if !reflect.DeepEqual(tt.list, tt.want) {
				t.Errorf("movingList.move() =\n%v\nwant\n%v", tt.list, tt.want)
			}
		})
	}
}

var benchListSizes []int

func init() {
	for i := 0; i <= 6; i++ {
		benchListSizes = append(benchListSizes, int(math.Pow10(i)))
	}
}

func BenchmarkList_Move(b *testing.B) {
	for _, bb := range benchListSizes {
		list := newMovingList(make([]int, bb))

		b.Run(strconv.Itoa(bb), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				list.move(i)
			}
		})
	}
}

func TestMovingAverage_Move(t *testing.T) {
	ma := MovingAverage{list: newMovingList([]float64{1.0, 2.0, 3.0})}
	want := MovingAverage{list: movingList[float64]{
		entries: []float64{4.0, 2.0, 3.0},
		pos:     1,
	}}

	if ma.Move(4.0); !reflect.DeepEqual(ma, want) {
		t.Errorf("MovingAverage.Avg() =\n%v\nwant\n%v", ma, want)
	}
}

func TestMovingAverage_sum(t *testing.T) {
	ma := MovingAverage{list: newMovingList([]float64{1.0, 2.0, 3.0})}
	const want = 6.0

	if got := ma.sum(); got != want {
		t.Errorf("MovingAverage.sum() = %v, want %v", got, want)
	}
}

func TestMovingAverage_Avg(t *testing.T) {
	ma := MovingAverage{list: newMovingList([]float64{1.0, 2.0, 3.0})}
	const want = 2.0

	if got := ma.Avg(); got != want {
		t.Errorf("MovingAverage.Avg() = %v, want %v", got, want)
	}
}

func BenchmarkTestMovingAverage_Avg(b *testing.B) {
	for _, bb := range benchListSizes {
		list := make([]float64, bb)

		for i := 0; i < bb; i++ {
			list[i] = float64(i)
		}

		ma := MovingAverage{list: newMovingList(list)}

		b.Run(strconv.Itoa(bb), func(b *testing.B) {
			ma.Avg()
		})
	}
}

func TestMovingAverage_AvgIncl(t *testing.T) {
	ma := MovingAverage{list: newMovingList([]float64{1.0, 2.0, 3.0})}

	tests := []struct {
		v      float64
		weight float64
		want   float64
	}{
		{
			4.0,
			1.0,
			2.5,
		},
		{
			4.0,
			0.5,
			8.0 / 3.5,
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprint(tt.v, tt.weight), func(t *testing.T) {
			if got := ma.AvgIncl(tt.v, tt.weight); got != tt.want {
				t.Errorf("MovingAverage.Avg() = %v, want %v", got, tt.want)
			}
		})
	}
}

func BenchmarkTestMovingAverage_AvgIncl(b *testing.B) {
	for _, bb := range benchListSizes {
		list := make([]float64, bb)

		for i := 0; i < bb; i++ {
			list[i] = float64(i)
		}

		ma := MovingAverage{list: newMovingList(list)}

		b.Run(strconv.Itoa(bb), func(b *testing.B) {
			ma.AvgIncl(4.0, 0.5)
		})
	}
}

func ExampleMovingAverage_AvgIncl() {
	ma := MovingAverage{list: newMovingList([]float64{1.0, 2.0, 3.0})}
	fmt.Println(ma.AvgIncl(4.0, 1.0))
	fmt.Println(ma.AvgIncl(4.0, 0.5))

	// Output: 2.5
	// 2.2857142857142856
}
