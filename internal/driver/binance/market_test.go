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
	"errors"
	"testing"

	"github.com/gorilla/schema"
	"github.com/muhlemmer/yatgo/internal/driver"
	"github.com/rs/zerolog"
)

func TestMarketData_GetJSON(t *testing.T) {
	logger := zerolog.New(zerolog.NewTestWriter(t))

	type args struct {
		ctx    context.Context
		path   string
		data   interface{}
		target interface{}
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			"time",
			args{
				logger.WithContext(testCTX),
				"/api/v3/time",
				nil,
				&ServerTimeResp{},
			},
			false,
		},
		{
			"wrong arg",
			args{
				logger.WithContext(testCTX),
				"/api/v3/time",
				"foo",
				&ServerTimeResp{},
			},
			true,
		},
		{
			"context error",
			args{
				logger.WithContext(errCTX),
				"/api/v3/time",
				nil,
				&ServerTimeResp{},
			},
			true,
		},
		{
			"not found error",
			args{
				logger.WithContext(testCTX),
				"/api/hello/world",
				nil,
				&ServerTimeResp{},
			},
			true,
		},
		{
			"order book, missing arg",
			args{
				logger.WithContext(testCTX),
				"/api/v3/depth",
				OrderBookReq{},
				&OrderBookResp{},
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &MarketData{
				Client: &driver.Client{
					Hosts: apiHosts,
				},
				se: schema.NewEncoder(),
			}
			err := m.GetJSON(tt.args.ctx, tt.args.path, tt.args.data, tt.args.target)
			logger.Err(err).Msg("")

			if (err != nil) != tt.wantErr {
				t.Errorf("MarketData.GetJSON() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				switch got := tt.args.target.(type) {
				case *ServerTimeResp:
					if got.ServerTime == 0 {
						t.Error("MarketData.GetJSON() ServerTime empty")
					}
				default:
					t.Fatal("no target defined")
				}
			}
		})
	}
}

func TestMarketData_GetJSON_backOff(t *testing.T) {
	logger := zerolog.New(zerolog.NewTestWriter(t))

	m := &MarketData{
		Client: &driver.Client{
			Hosts: apiHosts,
		},
		se: schema.NewEncoder(),
	}

	req := OrderBookReq{
		Symbol: "BTCUSDT",
		Limit:  OrderBookLimit_5000,
	}

	for {
		err := m.GetJSON(logger.WithContext(testCTX), "/api/v3/depth", req, &OrderBookResp{})
		logger.Err(err).Msg("")

		if err != nil {
			var boe BackOffError
			if !errors.As(err, &boe) {
				t.Fatalf("MarketData.GetJSON() error %v is of type %T expected type %T", err, err, boe)
			}
			break
		}
	}

	if err := m.GetJSON(logger.WithContext(testCTX), "/api/v3/depth", req, &OrderBookResp{}); err != nil {
		t.Fatal(err)
	}
}
