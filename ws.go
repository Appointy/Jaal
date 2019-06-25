package jaal

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/appointy/idgen"
	"github.com/gorilla/websocket"
	"gocloud.dev/pubsub"

	"go.appointy.com/jaal/graphql"
	"go.appointy.com/jaal/schemabuilder"
	"go.appointy.com/jaal/subscription"
)

// HTTPSubHandler implements the handler required for executing the graphql subscriptions
func HTTPSubHandler(schema *graphql.Schema, s *pubsub.Subscription) (http.Handler, func()) {
	source := make(chan *subscription.Event)
	sessions := &sessions{
		data: map[string]chan *subscription.Event{},
	}
	done := make(chan struct{})
	return &httpSubHandler{
			handler: handler{
				schema:   schema,
				executor: &graphql.Executor{},
			},
			qmHandler: HTTPHandler(schema),
			upgrader:  &websocket.Upgrader{},
			source:    source,
			done:      done,
		}, func() {
			go startListening(s, source, func() { close(done) })
			go listenSource(source, sessions)
		}
}

func listenSource(events chan *subscription.Event, ss *sessions) {
	for evt := range events {
		ss.RLock()
		for _, s := range ss.data {
			s <- evt
		}
		ss.RUnlock()
	}
}

func startListening(s *pubsub.Subscription, source chan<- *subscription.Event, cancel func()) {
	for {
		msg, err := s.Receive(context.Background())
		if err != nil {
			// TODO: Log
			cancel()
			return
		}
		msg.Ack()

		source <- &subscription.Event{
			Payload: msg.Body,
			Type:    msg.Metadata["type"],
		}
	}
}

type httpSubHandler struct {
	handler
	qmHandler http.Handler
	upgrader  *websocket.Upgrader
	source    chan *subscription.Event
	sessions  *sessions
	done      <-chan struct{}
}

type sessions struct {
	sync.RWMutex
	data map[string]chan *subscription.Event
}

func (h *httpSubHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet { // If not a subscription request route to normal handler
		h.qmHandler.ServeHTTP(w, r)
		return
	}

	if err := h.serveHTTP(w, r); err != nil {
		if err := writeResponse(w, nil, err); err != nil {
			fmt.Println(err)
		}
	}
}

func writeResponse(w io.Writer, value interface{}, err error) error {
	res := httpResponse{
		Data: value,
	}

	if err != nil {
		res.Errors = append(res.Errors, err.Error())
	}

	return json.NewEncoder(w).Encode(&res)
}

func (h *httpSubHandler) serveHTTP(w http.ResponseWriter, r *http.Request) error {
	var params map[string]interface{}
	if err := json.NewDecoder(strings.NewReader(strings.Trim(r.URL.Query().Get("variables"), "\""))).Decode(&params); err != nil {
		return err
	}

	query, err := graphql.Parse(r.URL.Query().Get("query"), params)
	if err != nil {
		return err
	}

	if len(query.SelectionSet.Selections) != 1 {
		return fmt.Errorf("exactly one subscription is expected")
	}

	schema := h.schema.Subscription
	if err := graphql.ValidateQuery(r.Context(), schema, query.SelectionSet); err != nil {
		return err
	}

	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return err
	}
	defer conn.Close()

	sid := idgen.New("sess")
	sess := make(chan *subscription.Event)
	h.sessions.Lock()
	h.sessions.data[sid] = sess
	h.sessions.Unlock()

	end := make(chan struct{}, 1)

	go func(conn *websocket.Conn) {
		_, _, _ = conn.ReadMessage()
		end <- struct{}{}
	}(conn)

	cls := func(ss *sessions, sid string) {
		ss.Lock()
		s := ss.data[sid]
		delete(ss.data, sid)
		ss.Unlock()
		close(s)
		for range s {
		}
	}

	// Listening on usrChannel for any source event of subType
	for msg := range sess {
		select {
		case <-end:
			cls(h.sessions, sid)
			return nil
		case <-h.done:
			cls(h.sessions, sid)
			return nil
		default:
			if err := func() error {
				res, err := h.executor.Execute(r.Context(), schema, &schemabuilder.Subscription{msg.Payload}, query)
				if err == ErrNoUpdate {
					return nil
				}
				rer := err
				w, err := conn.NextWriter(websocket.TextMessage)
				if err != nil {
					return err
				}
				defer w.Close()
				if err := writeResponse(w, res, rer); err != nil {
					return err
				}

				return nil
			}(); err != nil {
				// TODO: Log error
				cls(h.sessions, sid)
				return nil
			}
		}
	}

	return nil
}

var ErrNoUpdate = errors.New("don't update")
