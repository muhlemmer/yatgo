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
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/schema"
	"github.com/muhlemmer/yatgo/internal/driver"
)

var (
	// Global IP based back-off WaitGroup.
	// The WaitGroup will be blocked after any 429 or 418,
	// for the time set in the `Retry-After` reponse header.
	IPBackOff sync.WaitGroup
)

type MarketData struct {
	*driver.Client
	se *schema.Encoder
}

var apiHosts = []string{
	"api.binance.com",
	"api1.binance.com",
	"api2.binance.com",
	"api3.binance.com",
}

func (m *MarketData) encodeFormData(data interface{}) (url.Values, error) {
	if data == nil {
		return nil, nil
	}

	values := url.Values{}
	return values, m.se.Encode(data, values)
}

// BackOffError is returned after a 429 or 418 status code is received from the API.
type BackOffError struct {
	StatusCode int
	Duration   time.Duration
}

func (e BackOffError) Error() string {
	return fmt.Sprintf("binance: status %d, back off for %s", e.StatusCode, e.Duration)
}

// RequestError is returned on any status code that's not 200, 418 or 429.
type RequestError struct {
	StatusCode int
	Status     string
}

func (e RequestError) Error() string {
	return fmt.Sprintf("binance: status %s", e.Status)
}

// GetJSON performs a GET request on paths, with data encoded to URL values.
// The response body is expected to be JSON and will be unmarshalled into target.
// In case the call succeeds and the satus code is not 200, a BackOffError or RequestError will be returned.
//
// In case a status code 429 or 418 is received, a timer is started based on the 'Retry-After' response header.
// Subsequent calls will block untill this timer expires. (Uses the global IPBackOff WaitGroup)
func (m *MarketData) GetJSON(ctx context.Context, path string, data, target interface{}) error {
	values, err := m.encodeFormData(data)
	if err != nil {
		return fmt.Errorf("binance: %w", err)
	}

	IPBackOff.Wait()

	resp, err := m.Get(ctx, path, values)
	if err != nil {
		return fmt.Errorf("binance: %w", err)
	}

	if resp.StatusCode == 200 && resp.Body != nil {
		return json.NewDecoder(resp.Body).Decode(target)
	}

	if resp.StatusCode == 429 || resp.StatusCode == 418 {
		i, err := strconv.Atoi(resp.Header.Get("Retry-After"))
		if err != nil {
			return fmt.Errorf("binance Retry-After header: %w", err)
		}

		dt := time.Duration(i) * time.Second

		IPBackOff.Add(1)
		time.AfterFunc(dt, IPBackOff.Done)

		return BackOffError{
			StatusCode: resp.StatusCode,
			Duration:   dt,
		}
	}

	return RequestError{
		StatusCode: resp.StatusCode,
		Status:     resp.Status,
	}
}

type OrderBookLimit int

const (
	OrderBookLimit_5    OrderBookLimit = 5
	OrderBookLimit_10   OrderBookLimit = 10
	OrderBookLimit_20   OrderBookLimit = 20
	OrderBookLimit_50   OrderBookLimit = 50
	OrderBookLimit_100  OrderBookLimit = 100
	OrderBookLimit_500  OrderBookLimit = 500
	OrderBookLimit_1000 OrderBookLimit = 1000
	OrderBookLimit_5000 OrderBookLimit = 5000
	OrderBookDefault                   = OrderBookLimit_100
)

type OrderBookReq struct {
	Symbol string         `schema:"symbol,required,omitempty"`
	Limit  OrderBookLimit `schema:"limit,omitempty"`
}

type OrderBookResp struct {
	LastUpdateId int64      `json:"lastUpdateId"`
	Bids         [][]string `json:"bids"`
	Asks         [][]string `json:"asks"`
}

type PingResp struct{}

type ServerTimeResp struct {
	ServerTime int64 `json:"serverTime"`
}
