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
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog"
)

var (
	testCTX context.Context
	errCTX  context.Context
)

func testMain(m *testing.M) int {
	var cancel context.CancelFunc
	testCTX, cancel = context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	errCTX, cancel = context.WithCancel(testCTX)
	cancel()

	return m.Run()
}

func TestMain(m *testing.M) {
	os.Exit(testMain(m))
}

func TestDialWebsocket(t *testing.T) {
	logger := zerolog.New(zerolog.NewTestWriter(t))

	type args struct {
		ctx      context.Context
		endpoint string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			"context err",
			args{logger.WithContext(errCTX), "wss://stream.binance.com:9443/ws"},
			true,
		},
		{
			"address err",
			args{logger.WithContext(testCTX), "wss://example.com/ws"},
			true,
		},
		{
			"success",
			args{logger.WithContext(testCTX), "wss://stream.binance.com:9443/ws"},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ws, err := DialWebsocket(tt.args.ctx, websocket.DefaultDialer, tt.args.endpoint, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("DialWebsocket() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if ws != nil {
				defer func() {
					if err := ws.Close(); err != nil {
						t.Fatal(err)
					}
				}()
			}

			if !tt.wantErr {
				ctx, cancel := context.WithTimeout(tt.args.ctx, 5*time.Second)
				defer cancel()

				pong := make(chan struct{})
				ws.SetPongHandler(func(appData string) error {
					logger.Info().Str("appData", appData).Msg("pong receive")
					pong <- struct{}{}
					return nil
				})

				if err := ws.WriteControl(websocket.PingMessage, []byte("Hello, world!"), time.Now().Add(5*time.Second)); err != nil {
					t.Fatal(err)
				}

				ec := make(chan error, 1)

				go func() {
					for {
						if _, _, err := ws.NextReader(); err != nil {
							ec <- err
						}
					}
				}()

				select {
				case <-ctx.Done():
					t.Fatal(fmt.Errorf("pong wait: %w", ctx.Err()))
				case err := <-ec:
					t.Fatal(err)
				case <-pong:
				}
			}
		})
	}
}
