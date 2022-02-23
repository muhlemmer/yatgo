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

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"testing"

	"github.com/rs/zerolog"
)

func TestClient_tryRequest(t *testing.T) {
	logger := zerolog.New(zerolog.NewTestWriter(t))

	type args struct {
		ctx    context.Context
		method string
		u      url.URL
		body   io.Reader
	}
	tests := []struct {
		name           string
		Hosts          []string
		args           args
		wantStatusCode int
		wantErr        bool
	}{
		{
			"bad method",
			[]string{
				"api.binance.com",
				"api1.binance.com",
				"api2.binance.com",
				"api3.binance.com",
			},
			args{
				logger.WithContext(testCTX),
				"=",
				url.URL{
					Scheme: "https",
					Path:   "api/v3/ping",
				},
				nil,
			},
			0,
			true,
		},
		{
			"lookup failures",
			[]string{
				"tja",
				"foo",
				"bar",
			},
			args{
				logger.WithContext(testCTX),
				http.MethodGet,
				url.URL{
					Scheme: "https",
					Path:   "api/v3/ping",
				},
				nil,
			},
			0,
			true,
		},
		{
			"success",
			[]string{
				"api.binance.com",
				"api1.binance.com",
				"api2.binance.com",
				"api3.binance.com",
			},
			args{
				logger.WithContext(testCTX),
				http.MethodGet,
				url.URL{
					Scheme: "https",
					Path:   "api/v3/ping",
				},
				nil,
			},
			200,
			false,
		},
		{
			"success after fails",
			[]string{
				"tja",
				"foo",
				"bar",
				"api3.binance.com",
			},
			args{
				logger.WithContext(testCTX),
				http.MethodGet,
				url.URL{
					Scheme: "https",
					Path:   "api/v3/ping",
				},
				nil,
			},
			200,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Client{
				Hosts: tt.Hosts,
			}
			got, err := c.tryRequest(tt.args.ctx, tt.args.method, tt.args.u, tt.args.body)
			if (err != nil) != tt.wantErr {
				t.Errorf("Client.tryRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantStatusCode != 0 && got.StatusCode != tt.wantStatusCode {
				t.Errorf("Client.tryRequest() = %v, want %v", got.StatusCode, tt.wantStatusCode)
			}
		})
	}
}

func TestClient_Get(t *testing.T) {
	logger := zerolog.New(zerolog.NewTestWriter(t))

	type args struct {
		ctx    context.Context
		path   string
		values url.Values
	}
	tests := []struct {
		name           string
		hosts          []string
		args           args
		wantStatusCode int
		wantErr        bool
	}{
		{
			"lookup failures",
			[]string{
				"tja",
				"foo",
				"bar",
			},
			args{
				logger.WithContext(testCTX),
				"api/v3/ping",
				nil,
			},
			0,
			true,
		},
		{
			"context failures",
			[]string{
				"api.binance.com",
				"api1.binance.com",
				"api2.binance.com",
				"api3.binance.com",
			},
			args{
				logger.WithContext(errCTX),
				"api/v3/ping",
				nil,
			},
			0,
			true,
		},
		{
			"success",
			[]string{
				"api.binance.com",
				"api1.binance.com",
				"api2.binance.com",
				"api3.binance.com",
			},
			args{
				logger.WithContext(testCTX),
				"api/v3/ping",
				nil,
			},
			200,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Client{Hosts: tt.hosts}

			got, err := c.Get(tt.args.ctx, tt.args.path, tt.args.values)
			if (err != nil) != tt.wantErr {
				t.Errorf("Client.Get() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantStatusCode != 0 && got.StatusCode != tt.wantStatusCode {
				t.Errorf("Client.Get() = %v, want %v", got.StatusCode, tt.wantStatusCode)
			}
		})
	}
}
