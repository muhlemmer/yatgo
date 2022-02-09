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
	"reflect"
	"testing"

	"github.com/rs/zerolog"
)

func Test_handlerMap(t *testing.T) {
	var m handlerMap

	ss := []string{"a", "b", "c"}

	for _, key := range ss {
		m.Store(key, make(chan<- []byte))
	}

	for _, key := range ss {
		c, ok := m.Load(key)
		if !ok || c == nil {
			t.Fatal("handlerMap.Load() failed")
		}
	}

	m.Range(func(_ string, c chan<- []byte) bool {
		close(c)
		return true
	})
}

func TestStream_dispatch(t *testing.T) {
	logger := zerolog.New(zerolog.NewTestWriter(t))

	tests := []struct {
		name       string
		resp       streamResponse
		wantMethod wsMethodResponse
		wantStream []byte
	}{
		{
			"unhandeled",
			streamResponse{
				Stream: "spanac",
				Data:   json.RawMessage(`["Hello, World!"]`),
			},
			wsMethodResponse{},
			nil,
		},
		{
			"stream handler",
			streamResponse{
				Stream: "handler",
				Data:   json.RawMessage(`["Hello, World!"]`),
			},
			wsMethodResponse{},
			[]byte(`["Hello, World!"]`),
		},
		{
			"method response",
			streamResponse{
				ID:     1,
				Result: "Hello, World!",
			},
			wsMethodResponse{
				ID:     1,
				Result: "Hello, World!",
			},
			nil,
		},
		{
			"method response not found",
			streamResponse{
				ID:     99,
				Result: "Hello, World!",
			},
			wsMethodResponse{},
			nil,
		},
		{
			"unhandeled error",
			streamResponse{
				Error: &wsMethodError{
					Code: 3,
					Msg:  "foobar",
				},
			},
			wsMethodResponse{},
			nil,
		},
		{
			"error",
			streamResponse{
				Error: &wsMethodError{
					Code: 3,
					Msg:  "foobar",
				},
				ID: 1,
			},
			wsMethodResponse{
				ID: 1,
				Error: &wsMethodError{
					Code: 3,
					Msg:  "foobar",
				},
			},
			nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Stream{
				ctx: logger.WithContext(testCTX),
			}

			rc := make(chan wsMethodResponse, 1)
			s.addReponseChan(rc)

			handler := make(chan []byte, 1)
			s.handlers.Store("handler", handler)

			s.wg.Add(1)
			go s.dispatch(tt.resp)
			s.wg.Wait()

			close(rc)
			close(handler)

			if got := <-rc; !reflect.DeepEqual(got, tt.wantMethod) {
				t.Errorf("Stream.dispatch() method resp = %v, want %v", got, tt.wantMethod)
			}

			if got := <-handler; !reflect.DeepEqual(got, tt.wantStream) {
				t.Errorf("Stream.dispatch() stream resp = %s, want %s", got, tt.wantStream)
			}

		})
	}
}

func TestNewStream(t *testing.T) {
	logger := zerolog.New(zerolog.NewTestWriter(t))

	tests := []struct {
		name    string
		ctx     context.Context
		wantErr bool
	}{
		{
			"context error",
			logger.WithContext(errCTX),
			true,
		},
		{
			"success",
			logger.WithContext(testCTX),
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(tt.ctx)
			defer cancel()

			stream, err := NewStream(ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewStream() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				cancel()
				stream.wg.Wait()
			}
		})
	}
}

func TestStream_queue(t *testing.T) {
	ctx, cancel := context.WithCancel(testCTX)
	defer cancel()

	logger := zerolog.New(zerolog.NewTestWriter(t)).With().Timestamp().Logger()

	stream, err := NewStream(logger.WithContext(ctx))
	if err != nil {
		t.Fatal(err)
	}

	rc := stream.addQueue(wsMethodRequest{
		Method: "GET_PROPERTY",
		Params: []interface{}{"combined"},
	})

	want := wsMethodResponse{
		ID:     1,
		Result: true,
	}

	if got := <-rc; !reflect.DeepEqual(got, want) {
		t.Errorf("Stream method request = %v, want %v", got, want)
	}

	cancel()
	stream.wg.Wait()

	rc = stream.addQueue(wsMethodRequest{
		Method: "GET_PROPERTY",
		Params: []interface{}{"combined"},
	})

	if got := <-rc; got.Error == nil {
		t.Error("Stream method request expected error")
	}
}

func TestMethodRequest(t *testing.T) {
	ctx, cancel := context.WithCancel(testCTX)
	defer cancel()

	logger := zerolog.New(zerolog.NewTestWriter(t)).With().Timestamp().Logger()

	s, err := NewStream(logger.WithContext(ctx))
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		req     wsMethodRequest
		want    wsMethodResponse
		wantErr bool
	}{
		{
			wsMethodRequest{
				Method: MethodWsSubscribe,
				Params: []interface{}{
					"btcusdt@aggTrade",
					"btcusdt@depth",
				},
			},
			wsMethodResponse{
				ID: 1,
			},
			false,
		},
		{
			wsMethodRequest{
				Method: MethodWsUnsubscribe,
				Params: []interface{}{
					"btcusdt@depth",
				},
			},
			wsMethodResponse{
				ID: 2,
			},
			false,
		},
		{
			wsMethodRequest{
				Method: MethodWsListSubscriptions,
			},
			wsMethodResponse{
				ID: 3,
				Result: []interface{}{
					"btcusdt@aggTrade",
				},
			},
			false,
		},
		{
			wsMethodRequest{
				Method: MethodWsSetProperty,
				Params: []interface{}{
					"combined",
					true,
				},
			},
			wsMethodResponse{
				ID: 4,
			},
			false,
		},
		{
			wsMethodRequest{
				Method: MethodWsGetProperty,
				Params: []interface{}{
					"combined",
				},
			},
			wsMethodResponse{
				ID:     5,
				Result: true,
			},
			false,
		},
		{
			wsMethodRequest{
				Method: MethodWsGetProperty,
				Params: []interface{}{
					"spanac",
				},
			},
			wsMethodResponse{
				ID: 6,
				Error: wsMethodError{
					Code: 0,
					Msg:  "Unknown property",
				},
			},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.req.Method, func(t *testing.T) {
			got := <-s.addQueue(tt.req)

			if (got.Error != nil) != tt.wantErr {
				t.Errorf("Stream method response Err = %v, wantErr %v", got.Error, tt.wantErr)
			}

			if got.ID != tt.want.ID || !reflect.DeepEqual(got.Result, tt.want.Result) {
				t.Errorf("Stream method response = %v, want %v", got, tt.want)
			}

		})
	}

	cancel()
	s.wg.Wait()
}
