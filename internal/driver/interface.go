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

package driver

type ClosingPrice struct {
	Price  float64
	Closed bool // Period is fininshed
}

type ClosingPriceHandler interface {
	Event(ClosingPrice)
	Done()
}

type ClosingPriceStreamer interface {
	SubscribeClosingPrices(symbol string, interval string, handler ClosingPriceHandler) error
	UnsubscribeClosingPrices(symbol string, interval string) error
}
