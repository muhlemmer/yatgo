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
	"errors"
	"fmt"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/muhlemmer/yatgo/internal/driver"
	"github.com/rs/zerolog"
	"go.uber.org/ratelimit"
)

type wsMethodError struct {
	Code int
	Msg  string
}

func (e wsMethodError) Error() string {
	return fmt.Sprintf("binance websocket response code %d, %s", e.Code, e.Msg)
}

type wsMethodResponse struct {
	ID     uint
	Result interface{}
	Error  error
}

type wsMethodRequest struct {
	Method string        `json:"method,omitempty"`
	Params []interface{} `json:"params,omitempty"`

	// Do not set below this line,
	// handled by the queue manager!
	ID uint `json:"id,omitempty"`
}

// Stream implements the binance cobined stream protocol.
type Stream struct {
	ctx    context.Context
	cancel context.CancelFunc

	conn     *websocket.Conn
	handlers driver.SyncMap[string, driver.JSONHandler]
	wg       sync.WaitGroup

	queue  chan wsMethodRequest
	qlimit ratelimit.Limiter
	qmtx   sync.Mutex
	qid    uint
	qrc    map[uint]chan<- wsMethodResponse
}

type streamMessage struct {
	Error *wsMethodError `json:"error,omitempty"`

	// method response
	ID     uint        `json:"id,omitempty"`
	Result interface{} `json:"result,omitempty"`

	// stream events
	Stream string          `json:"stream,omitempty"`
	Data   json.RawMessage `json:"data,omitempty"`
}

func (s *Stream) listen() {
	defer s.wg.Done()
	defer s.cancel()

	for {
		_, data, err := s.conn.ReadMessage()
		if err != nil {
			zerolog.Ctx(s.ctx).Err(err).Msg("websocket receive")
			return
		}

		s.wg.Add(1)
		go s.dispatch(data)
	}
}

func (s *Stream) popResponseChan(id uint) (rc chan<- wsMethodResponse, ok bool) {
	s.qmtx.Lock()
	rc, ok = s.qrc[id]
	if ok {
		delete(s.qrc, id)
	}
	s.qmtx.Unlock()

	return rc, ok
}

func (s *Stream) dispatch(data []byte) {
	defer s.wg.Done()

	logger := zerolog.Ctx(s.ctx).With().RawJSON("data", data).Logger()
	logger.Debug().Msg("")

	defer func() {
		x := recover()
		if x != nil {
			err, _ := x.(error)
			if err == nil {
				logger.Panic().Interface("value", x).Msg("re-panic in dispatch recover")
				return
			}

			logger.Err(err).Msg("dispatch panic recover")
		}
	}()

	var msg streamMessage
	err := json.Unmarshal(data, &msg)

	if err != nil {
		panic(fmt.Errorf("stream.dispatch: %w", err))
	}

	if msg.Error != nil {
		if msg.ID != 0 {
			s.sendErrResponse(msg.ID, msg.Error)
		} else {
			logger.Err(msg.Error).Msg("protocol error in dispatch")
		}

		return
	}

	if msg.ID != 0 {
		if c, ok := s.popResponseChan(msg.ID); ok {
			c <- wsMethodResponse{
				ID:     msg.ID,
				Result: msg.Result,
			}
		} else {
			logger.Warn().Msg("unknown request ID in method response dispatch")
		}
		return
	}

	if msg.Stream != "" {
		if handler, ok := s.handlers.Load(msg.Stream); ok {
			handler.Event(msg.Data)
			return
		}
	}

	logger.Warn().Msg("unhandeled message in dispatch")
}

func (s *Stream) addReponseChan(rc chan<- wsMethodResponse) (id uint) {
	s.qmtx.Lock()
	defer s.qmtx.Unlock()

	if s.qrc == nil {
		s.qrc = make(map[uint]chan<- wsMethodResponse)
	}

	s.qid++
	s.qrc[s.qid] = rc

	return s.qid
}

func (s *Stream) addQueue(msg wsMethodRequest) <-chan wsMethodResponse {
	rc := make(chan wsMethodResponse, 1)

	if s.ctx.Err() != nil {
		rc <- wsMethodResponse{Error: websocket.ErrCloseSent}
		return rc
	}

	msg.ID = s.addReponseChan(rc)

	s.queue <- msg
	return rc
}

func (s *Stream) sendErrResponse(reqID uint, err error) {
	rc, ok := s.popResponseChan(reqID)

	if ok {
		rc <- wsMethodResponse{
			ID:    reqID,
			Error: err,
		}
	}
}

func (s *Stream) close() {
	s.cancel()
	close(s.queue)

	err := s.conn.Close()
	zerolog.Ctx(s.ctx).Err(err).Msg("stream closed")

	// drain the channel
	for msg := range s.queue {
		s.sendErrResponse(msg.ID, err)
	}

	s.handlers.Range(func(_ string, handler driver.JSONHandler) bool {
		handler.Done()
		return true
	})
}

func (s *Stream) sendQueue() {
	defer s.wg.Done()

	var err error

work:
	for {

		select {
		case <-s.ctx.Done():
			break work
		case msg := <-s.queue:
			s.qlimit.Take()

			if s.ctx.Err() != nil {
				break work
			}

			err = s.conn.WriteJSON(msg)
			zerolog.Ctx(s.ctx).Err(err).Interface("msg", msg).Msg("websocket send")

			if err != nil {
				err = fmt.Errorf("binance stream send: %w", err)
				s.sendErrResponse(msg.ID, err)
				break work
			}
		}
	}

	s.close()
}

var newStreamLimiter = ratelimit.New(5)

// NewStream dails the websocket endpoint for binance combined streams.
// The returned stream is closed when the context is canceled.
// On any error, the stream closes and terminates.
// Calling methods on the Stream after closingwill results in errors to be returned.
func NewStream(ctx context.Context) (*Stream, error) {
	logger := zerolog.Ctx(ctx).With().Str("driver", "binance").Str("obj", "Stream").Logger()
	ctx = logger.WithContext(ctx)

	newStreamLimiter.Take()

	conn, err := driver.DialWebsocket(ctx, websocket.DefaultDialer, EndpointWsStream, nil)
	if err != nil {
		return nil, fmt.Errorf("binance.NewStream: %w", err)
	}

	s := &Stream{
		conn:   conn,
		queue:  make(chan wsMethodRequest, 64),
		qlimit: ratelimit.New(5),
	}

	s.ctx, s.cancel = context.WithCancel(ctx)

	s.wg.Add(2)
	go s.listen()
	go s.sendQueue()

	return s, nil
}

var (
	ErrStreamSubscribed = errors.New("stream already subscribed")
)

// Subscribe to a named binanace websocket stream.
// Raw JSON will be send to the returned channel for every complete message.
// The order of messages is serialized in order of arrival,
// and the Stream's listener will block untill the channel write completes.
// The receiver must prevent exessive blocking of the channel.
// The channel can be buffered with the size of bufLen,
// to accomodate for short bursts of data.
func (s *Stream) Subscribe(stream string, handler driver.JSONHandler) error {
	if _, loaded := s.handlers.LoadOrStore(stream, handler); loaded {
		return ErrStreamSubscribed
	}

	resp := <-s.addQueue(wsMethodRequest{
		Method: MethodWsSubscribe,
		Params: []interface{}{stream},
	})

	if resp.Error != nil {
		s.handlers.Delete(stream)
		return fmt.Errorf("stream.Subscribe: %w", resp.Error)
	}

	return nil
}

func (s *Stream) Unsubscribe(stream string) error {
	resp := <-s.addQueue(wsMethodRequest{
		Method: MethodWsUnsubscribe,
		Params: []interface{}{stream},
	})

	if resp.Error != nil {
		return fmt.Errorf("stream.Unsubscribe: %w", resp.Error)
	}

	if handler, ok := s.handlers.LoadAndDelete(stream); ok {
		handler.Done()
	}

	return nil
}
