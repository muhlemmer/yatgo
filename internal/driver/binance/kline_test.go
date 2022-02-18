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

package binance

import (
	"reflect"
	"testing"

	"github.com/muhlemmer/yatgo/internal/driver"
)

type testKlineHandler struct {
	got chan KlineEvent
}

func (h testKlineHandler) Event(event KlineEvent) {
	h.got <- event
}

func (h testKlineHandler) Done() {
	close(h.got)
}

func newTestKlineHandler(bufLen int) testKlineHandler {
	return testKlineHandler{
		got: make(chan KlineEvent, bufLen),
	}
}

func Test_klineHandler_Event(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		want    KlineEvent
		wantErr bool
	}{
		{
			"success",
			`{
				"e": "kline",
				"E": 123456789,
				"s": "BTCUSDT",
				"k": {
				  "t": 123400000,
				  "T": 123460000,
				  "s": "BTCUSDT",
				  "i": "1m",
				  "f": 100,
				  "L": 200,
				  "o": "0.0010",
				  "c": "0.0020",
				  "h": "0.0025",
				  "l": "0.0015",
				  "v": "1000",
				  "n": 100,
				  "x": false,
				  "q": "1.0000",
				  "V": "500",
				  "Q": "0.500",
				  "B": "123456"
				}
			  }`,
			KlineEvent{
				Event:  "kline",
				Time:   123456789,
				Symbol: "BTCUSDT",
				Kline: Kline{
					Start:            123400000,
					Finish:           123460000,
					Symbol:           "BTCUSDT",
					Interval:         "1m",
					First:            100,
					Last:             200,
					Open:             "0.0010",
					Close:            "0.0020",
					High:             "0.0025",
					Low:              "0.0015",
					BaseVolume:       "1000",
					Trades:           100,
					Closed:           false,
					QuoteVolume:      "1.0000",
					TakerBaseVolume:  "500",
					TakerQuoteVolume: "0.500",
					Ignore:           "123456",
				},
			},
			false,
		},
		{
			"json error",
			`~`,
			KlineEvent{},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k := newTestKlineHandler(1)
			h := klineHandler{h: k}

			defer func() {
				if err, _ := recover().(error); (err != nil) != tt.wantErr {
					t.Errorf("klineHandler.Event() error = %v, wantErr %v", err, tt.wantErr)
				}
			}()

			h.Event([]byte(tt.data))
			h.h.Done()

			if got := <-k.got; !reflect.DeepEqual(got, tt.want) {
				t.Errorf("klineHandler.Event() = \n%v\nwant\n%v", got, tt.want)
			}
		})
	}
}

func TestSubscribeKlines(t *testing.T) {
	h := newTestKlineHandler(100)

	if err := testStream.SubscribeKlines("btcusdt", Minute, h); err != nil {
		t.Fatal(err)
	}

	select {
	case <-h.got:
	case <-testCTX.Done():
		t.Error("SubscribeKlines: no data received")
	}

	if err := testStream.UnsubscribeKlines("btcusdt", Minute); err != nil {
		t.Fatal(err)
	}

	for range h.got {
	}
}

type testClosingPriceHandler struct {
	got chan driver.ClosingPrice
}

func (h testClosingPriceHandler) Event(event driver.ClosingPrice) {
	h.got <- event
}

func (h testClosingPriceHandler) Done() {
	close(h.got)
}

func newTestClosingPriceHandler(bufLen int) testClosingPriceHandler {
	return testClosingPriceHandler{
		got: make(chan driver.ClosingPrice, bufLen),
	}
}

func Test_closingPriceHandler_Event(t *testing.T) {
	tests := []struct {
		name    string
		event   KlineEvent
		want    driver.ClosingPrice
		wantErr bool
	}{
		{
			"success",
			KlineEvent{
				Kline: Kline{
					Close:  "1.1",
					Closed: true,
				},
			},
			driver.ClosingPrice{
				Price:  1.1,
				Closed: true,
			},
			false,
		},
		{
			"error",
			KlineEvent{
				Kline: Kline{
					Close:  "foo",
					Closed: true,
				},
			},
			driver.ClosingPrice{},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k := newTestClosingPriceHandler(1)
			h := closingPriceHandler{h: k}

			defer func() {
				if err, _ := recover().(error); (err != nil) != tt.wantErr {
					t.Errorf("closingPriceHandler.Event() error = %v, wantErr %v", err, tt.wantErr)
				}
			}()

			h.Event(tt.event)
			h.h.Done()

			if got := <-k.got; !reflect.DeepEqual(got, tt.want) {
				t.Errorf("closingPriceHandler.Event() = \n%v\nwant\n%v", got, tt.want)
			}
		})
	}
}

func TestSubscribeClosingPrices(t *testing.T) {
	h := newTestClosingPriceHandler(100)

	if err := testStream.SubscribeClosingPrices("btcusdt", "1m", h); err != nil {
		t.Fatal(err)
	}

	select {
	case <-h.got:
	case <-testCTX.Done():
		t.Error("SubscribeClosingPrices: no data received")
	}

	if err := testStream.UnsubscribeClosingPrices("btcusdt", "1m"); err != nil {
		t.Fatal(err)
	}

	for range h.got {
	}
}
