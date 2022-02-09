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

// Package binance provides the connection driver for the binance API.
package binance

// Endpoint paths
const (
	EndpointWsBase   = "wss://stream.binance.com:9443"
	EndpointWsRaw    = EndpointWsBase + "/ws"
	EndpointWsStream = EndpointWsBase + "/stream"
)

// Method names for websocket
const (
	MethodWsSubscribe         = "SUBSCRIBE"
	MethodWsUnsubscribe       = "UNSUBSCRIBE"
	MethodWsListSubscriptions = "LIST_SUBSCRIPTIONS"
	MethodWsSetProperty       = "SET_PROPERTY"
	MethodWsGetProperty       = "GET_PROPERTY"
)
