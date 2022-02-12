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
	"reflect"
	"testing"
	"time"

	"github.com/rs/zerolog"
)

type testHandler struct {
	ctx    context.Context
	stream string
	events chan []byte
}

func newTestHandler(ctx context.Context, stream string, bufLen int) *testHandler {
	return &testHandler{
		ctx:    ctx,
		stream: stream,
		events: make(chan []byte, bufLen),
	}
}

func (h *testHandler) Event(data []byte) {
	zerolog.Ctx(h.ctx).Debug().RawJSON("data", data).Str("stream", h.stream).Msg("testHandler")
	h.events <- data
}

func (h *testHandler) Done() {
	zerolog.Ctx(h.ctx).Info().Msg("testHandler Done")
	close(h.events)
}

type panicHandler struct{}

func (panicHandler) Event([]byte) { panic("foo") }
func (panicHandler) Done()        {}

func TestStream_dispatch(t *testing.T) {
	logger := zerolog.New(zerolog.NewTestWriter(t))

	tests := []struct {
		name       string
		data       string
		wantMethod wsMethodResponse
		wantStream []byte
	}{
		{
			"unhandeled",
			`{"stream":"spanac","data":["Hello, World!"]}`,
			wsMethodResponse{},
			nil,
		},
		{
			"stream handler",
			`{"stream":"handler","data":["Hello, World!"]}`,
			wsMethodResponse{},
			[]byte(`["Hello, World!"]`),
		},
		{
			"method response",
			`{"id":1,"result":"Hello, World!"}`,
			wsMethodResponse{
				ID:     1,
				Result: "Hello, World!",
			},
			nil,
		},
		{
			"method response not found",
			`{"id":99,"result":"Hello, World!"}`,
			wsMethodResponse{},
			nil,
		},
		{
			"unhandeled error",
			`{"error":{"Code":3,"Msg":"foobar"}}`,
			wsMethodResponse{},
			nil,
		},
		{
			"error",
			`{"error":{"Code":3,"Msg":"foobar"},"id":1}`,
			wsMethodResponse{
				ID: 1,
				Error: &wsMethodError{
					Code: 3,
					Msg:  "foobar",
				},
			},
			nil,
		},
		{
			"recover",
			`!`,
			wsMethodResponse{},
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

			handler := newTestHandler(s.ctx, "dispatch_test", 1)

			s.handlers.Store("handler", handler)

			s.wg.Add(1)
			go s.dispatch([]byte(tt.data))
			s.wg.Wait()

			close(rc)
			handler.Done()

			if got := <-rc; !reflect.DeepEqual(got, tt.wantMethod) {
				t.Errorf("Stream.dispatch() method resp = %v, want %v", got, tt.wantMethod)
			}

			gotData := <-handler.events
			if !reflect.DeepEqual(gotData, tt.wantStream) {
				t.Errorf("Stream.dispatch() stream resp = %s, want %s", gotData, tt.wantStream)
			}

		})
	}

	t.Run("panic", func(t *testing.T) {
		s := &Stream{
			ctx: logger.WithContext(testCTX),
		}

		s.handlers.Store("handler", panicHandler{})

		defer func() {
			if recover() == nil {
				t.Error("recover returned nil")
			}
		}()

		s.wg.Add(1)
		s.dispatch([]byte(`{"stream":"handler","data":["Hello, World!"]}`))
	})
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

func TestStream_Subscribe(t *testing.T) {
	ctx, cancel := context.WithCancel(testCTX)
	defer cancel()

	logger := zerolog.New(zerolog.NewTestWriter(t)).With().Timestamp().Logger()

	s, err := NewStream(logger.WithContext(ctx))
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		stream  string
		wantErr bool
	}{
		{
			"btcusdt@aggTrade",
			false,
		},
		{
			"btcusdt@aggTrade",
			true,
		},
		{
			"",
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.stream, func(t *testing.T) {
			handler := newTestHandler(logger.WithContext(ctx), tt.stream, 100)

			err := s.Subscribe(tt.stream, handler)
			if (err != nil) != tt.wantErr {
				t.Errorf("Stream.Subscribe() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				select {
				case event := <-handler.events:
					if event != nil {
						return // success
					}
				case <-time.After(5 * time.Second):
					// time-out
				}

				if <-handler.events == nil {
					t.Fatal("no data received")
				}
			}
		})
	}

	s.cancel()
	s.wg.Wait()
}

func TestStream_Unsubscribe(t *testing.T) {
	ctx, cancel := context.WithCancel(testCTX)
	defer cancel()

	logger := zerolog.New(zerolog.NewTestWriter(t)).With().Timestamp().Logger()

	s, err := NewStream(logger.WithContext(ctx))
	if err != nil {
		t.Fatal(err)
	}

	const stream = "btcusdt@aggTrade"
	handler := newTestHandler(logger.WithContext(ctx), stream, 100)

	if err := s.Subscribe(stream, handler); err != nil {
		t.Fatal(err)
	}

	if err = s.Unsubscribe(stream); err != nil {
		t.Fatal(err)
	}

	if err = s.Unsubscribe(stream); err != nil {
		t.Fatal(err)
	}

	s.cancel()
	s.wg.Wait()
}
