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
