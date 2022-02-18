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
	"encoding/json"
	"fmt"
)

type KlineInterval string

const (
	Minute   KlineInterval = "1m"
	Minute3  KlineInterval = "3m"
	Minute5  KlineInterval = "5m"
	Minute15 KlineInterval = "15m"
	Minute30 KlineInterval = "30m"
	Hour     KlineInterval = "1h"
	Hour2    KlineInterval = "2h"
	Hour4    KlineInterval = "4h"
	Hour6    KlineInterval = "6h"
	Hour8    KlineInterval = "8h"
	Hour12   KlineInterval = "12h"
	Day      KlineInterval = "1d"
	Day3     KlineInterval = "3d"
	Week     KlineInterval = "1w"
	Month    KlineInterval = "1M"
)

type Kline struct {
	Start            int64  `json:"t"` // Kline start time
	Finish           int64  `json:"T"` // Kline close time
	Symbol           string `json:"s"` // Symbol
	Interval         string `json:"i"` // Interval
	First            int64  `json:"f"` // First trade ID
	Last             int64  `json:"L"` // Last trade ID
	Open             string `json:"o"` // Open price
	Close            string `json:"c"` // Close price
	High             string `json:"h"` // High price
	Low              string `json:"l"` // Low price
	BaseVolume       string `json:"v"` // Base asset volume
	Trades           int    `json:"n"` // Number of trades
	Closed           bool   `json:"x"` // Is this kline closed?
	QuoteVolume      string `json:"q"` // Quote asset volume
	TakerBaseVolume  string `json:"V"` // Taker buy base asset volume
	TakerQuoteVolume string `json:"Q"` // Taker buy quote asset volume
	Ignore           string `json:"B"` // Ignore
}

type KlineEvent struct {
	Event  string `json:"e"` // Event type ("kline")
	Time   int64  `json:"E"` // Event time
	Symbol string `json:"s"` // Symbol
	Kline  Kline  `json:"k"`
}

type klineHandler struct {
	h KlineHandler
}

func (k *klineHandler) Event(data []byte) {
	var event KlineEvent
	if err := json.Unmarshal(data, &event); err != nil {
		panic(fmt.Errorf("KlineHandler: %w", err))
	}

	k.h.Event(event)
}

func (k *klineHandler) Done() { k.h.Done() }

type KlineHandler interface {
	Event(KlineEvent)
	Done()
}

func klineStreamName(symbol string, interval KlineInterval) string {
	return fmt.Sprintf("%s@kline_%s", symbol, interval)
}

func (s *Stream) SubscribeKlines(symbol string, interval KlineInterval, handler KlineHandler) error {
	return s.Subscribe(
		klineStreamName(symbol, interval),
		&klineHandler{handler},
	)
}

func (s *Stream) UnsubscribeKlines(symbol string, interval KlineInterval) error {
	return s.Unsubscribe(klineStreamName(symbol, interval))
}
