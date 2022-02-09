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
	"os"
	"testing"
	"time"

	"github.com/rs/zerolog/log"
)

var (
	testCTX    context.Context
	errCTX     context.Context
	testStream *Stream
)

func testMain(m *testing.M) int {
	var cancel context.CancelFunc
	testCTX, cancel = context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	errCTX, cancel = context.WithCancel(testCTX)
	cancel()

	var err error
	testStream, err = NewStream(testCTX)
	if err != nil {
		log.Fatal().Err(err).Msg("testMain")
	}

	return m.Run()
}

func TestMain(m *testing.M) {
	os.Exit(testMain(m))
}
