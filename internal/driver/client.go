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
	"io"
	"net/http"
	"net/url"

	"github.com/rs/zerolog"
)

// Client wraps an HTTP client and provides
// request retries on fallback hosts.
type Client struct {
	http.Client
	Hosts []string
}

func (c *Client) tryRequest(ctx context.Context, method string, u url.URL, body io.Reader) (resp *http.Response, err error) {
	for _, ep := range c.Hosts {

		u.Host = ep
		logger := zerolog.Ctx(ctx).With().Stringer("url", &u).Logger()

		req, re := http.NewRequestWithContext(ctx, method, u.String(), body)
		if re != nil {
			return nil, fmt.Errorf("client Get: %w", err)
		}

		resp, err = c.Client.Do(req)

		if resp != nil {
			logger = logger.With().Str("status", resp.Status).Int64("content-length", resp.ContentLength).Logger()

			le := logger.Debug()
			for k, v := range resp.Header {
				le = le.Strs(k, v)
			}
			le.Msg("client response headers")
		}

		logger.Err(err).Msg("client Get")

		// In case of a connection or server-side error,
		// we are just going to retry the next end-point.
		if err == nil && resp.StatusCode < 500 {
			break
		}
	}

	return resp, err
}

// Get performs a HTTP request against all configured hosts, using path and URL encoded values.
// It returns after the first successfull call produces a status code <500.
// All status codes <500 are considered success and should be handeled by the caller.
func (c *Client) Get(ctx context.Context, path string, values url.Values) (resp *http.Response, err error) {
	return c.tryRequest(ctx, http.MethodGet, url.URL{
		Scheme:   "https",
		Path:     path,
		RawQuery: values.Encode(),
	}, nil)
}
