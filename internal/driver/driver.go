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

// Package driver provides primitives for exchange connection drivers.
package driver

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog"
)

func DialWebsocket(ctx context.Context, dialer *websocket.Dialer, endpoint string, requestHeader http.Header) (*websocket.Conn, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	logger := zerolog.Ctx(ctx).With().Str("endpoint", endpoint).Logger()

	conn, resp, err := websocket.DefaultDialer.DialContext(ctx, endpoint, requestHeader)

	if resp != nil {
		body, _ := ioutil.ReadAll(resp.Body)
		logger = logger.With().Str("status", resp.Status).Bytes("body", body).Logger()
	}
	logger.Err(err).Msg("driver.DialWebsocket")

	if err != nil {
		return nil, fmt.Errorf("driver.DialWebsocket: %w", err)
	}

	return conn, nil
}

// JSONStreamHandler handels incomming JSON messages on a websocket.
type JSONHandler interface {
	// Event is called on each complete JSON message.
	// Panics during execution must not infuence the socket listener.
	Event(data []byte)

	// Done is called when the orignating stream is closed or unsubscribed.
	// Handlers should expect Event calls untill Done is called,
	// even after unsubscribing to a stream.
	Done()
}

// SyncMap is a type-safe generic wrapper of sync.Map
type SyncMap[K, V any] struct {
	sync.Map
}

func (m *SyncMap[K, V]) Store(key K, value V) { m.Map.Store(key, value) }

func (m *SyncMap[K, V]) Load(key K) (value V, ok bool) {
	x, _ := m.Map.Load(key)
	value, ok = x.(V)
	return value, ok
}

func (m *SyncMap[K, V]) LoadOrStore(key K, value V) (actual V, loaded bool) {
	x, loaded := m.Map.LoadOrStore(key, value)
	actual = x.(V)

	return actual, loaded
}

func (m *SyncMap[K, V]) LoadAndDelete(key K) (value V, ok bool) {
	x, _ := m.Map.LoadAndDelete(key)
	value, ok = x.(V)
	return value, ok
}

func (m *SyncMap[K, V]) Range(f func(key K, value V) bool) {
	m.Map.Range(func(key, value any) bool {
		return f(key.(K), value.(V))
	})
}
