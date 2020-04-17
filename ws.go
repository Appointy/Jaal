package jaal

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/appointy/idgen"
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
	"gocloud.dev/pubsub"

	"go.saastack.io/jaal/graphql"
	"go.saastack.io/jaal/jerrors"
	"go.saastack.io/jaal/schemabuilder"
)

// HTTPSubHandler implements the handler required for executing the graphql subscriptions
func HTTPSubHandler(schema *graphql.Schema, subs *pubsub.Subscription) (http.Handler, func()) {
	source := make(chan *event)
	sessions := &sessions{
		data:  map[string]chan *event{},
		chans: map[string]chan struct{}{},
	}
	return &httpSubHandler{
			handler: handler{
				schema:   schema,
				executor: &graphql.Executor{},
			},
			qmHandler: HTTPHandler(schema),
			upgrader:  &websocket.Upgrader{},
			source:    source,
			sessions:  sessions,
		}, func() {
			go startListening(subs, source, func() {
				exit(sessions)
			})
			go listenSource(source, sessions)
		}
}

func listenSource(events chan *event, ss *sessions) {
	for evt := range events {
		ss.RLock()
		for _, v := range ss.data {
			v <- evt
		}
		ss.RUnlock()
	}
}

func startListening(s *pubsub.Subscription, source chan<- *event, cancel func()) {
	for {
		msg, err := s.Receive(context.Background())
		if err != nil {
			fmt.Println("Pubsub failed with error:", err)
			fmt.Println("Closing all sessions")
			cancel()
			return
		}
		msg.Ack()

		source <- &event{
			payload: msg.Body,
			typ:     msg.Metadata["type"],
		}
	}
}

type httpSubHandler struct {
	handler
	qmHandler http.Handler
	upgrader  *websocket.Upgrader
	source    chan *event
	sessions  *sessions
}

type event struct {
	typ     string
	payload []byte
}

type sessions struct {
	sync.RWMutex
	data  map[string]chan *event
	chans map[string]chan struct{}
}

type wsMessage struct {
	Type    string          `json:"type"`
	Id      string          `json:"id,omitempty"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

type gqlPayload struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables"`
	OpName    string                 `json:"operationName"`
}

func (h *httpSubHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet { // If not a subscription request route to normal handler
		h.qmHandler.ServeHTTP(w, r)
		return
	}
	log.Println("Request Headers:", r.Header)

	// Check origin and set response headers
	h.upgrader.CheckOrigin = func(r *http.Request) bool { return true }
	res := http.Header{}
	res["Sec-Websocket-Protocol"] = []string{"graphql-ws"}

	con, err := h.upgrader.Upgrade(w, r, res)
	if err != nil {
		fmt.Println("failed to upgrade to websocket:", err)
		return
	}
	defer con.Close()

	if con.Subprotocol() != "graphql-ws" {
		fmt.Println("invalid subprotocol")
		return
	}

	var msg wsMessage

	if err := con.ReadJSON(&msg); err != nil {
		fmt.Println("failed to parse websocket message: ", err)
		return
	}
	conn := &webConn{conn: con}
	if msg.Type != "connection_init" {
		if err := writeResponse(conn, "connection_error", "", nil, errors.New("expected init message")); err != nil {
			fmt.Println(err)
			return
		}
	}
	id := idgen.New("sock")
	if err := writeResponse(conn, "connection_ack", id, nil, nil); err != nil {
		fmt.Println(err)
		return
	}
loop:
	for {
		var data wsMessage
		if err := con.ReadJSON(&data); err != nil {
			if err := writeResponse(conn, "connection_error", "", nil, err); err != nil {
				fmt.Println(err)
				return
			}
			fmt.Println(err)
		}
		data.Id = id
		switch data.Type {
		case "start":
			var gql gqlPayload
			if err := json.Unmarshal(data.Payload, &gql); err != nil {
				if err := writeResponse(conn, "connection_error", data.Id, nil, err); err != nil {
					fmt.Println(err)
					return
				}
				fmt.Println(err)
				return
			}
			query, err := graphql.Parse(gql.Query, gql.Variables)
			if err != nil {
				if er := writeResponse(conn, "error", data.Id, nil, err); er != nil {
					fmt.Println(err)
					return
				}
				fmt.Println(err)
				return
			}
			schema := h.schema.Subscription
			if err := graphql.ValidateQuery(r.Context(), schema, query.SelectionSet); err != nil {
				if er := writeResponse(conn, "error", data.Id, nil, err); er != nil {
					fmt.Println(er)
					return
				}
				fmt.Println(err)
				return
			}
			for _, v := range query.SelectionSet.Selections {
				end := make(chan struct{}, 1)
				modQuery := &graphql.Query{
					Name: query.Name,
					Kind: query.Kind,
					SelectionSet: &graphql.SelectionSet{
						Selections: []*graphql.Selection{v},
						Fragments:  query.SelectionSet.Fragments,
					},
				}
				go func(conn *webConn, data *wsMessage, schema graphql.Type, query *graphql.Query, end chan struct{}, w http.ResponseWriter, r *http.Request) {
					err := h.serveHTTP(conn, *data, schema, query, end, w, r)
					if err := writeResponse(conn, "complete", data.Id, nil, err); err != nil {
						fmt.Println(err)
					}
					h.sessions.Lock()
					delete(h.sessions.data, data.Id)
					delete(h.sessions.chans, data.Id)
					h.sessions.Unlock()
				}(conn, &data, schema, modQuery, end, w, r)
			}
		case "stop":
			h.sessions.RLock()
			if _, ok := h.sessions.chans[data.Id]; ok {
				h.sessions.chans[data.Id] <- struct{}{}
			}
			h.sessions.RUnlock()
		case "connection_terminate":
			h.sessions.RLock()
			delete(h.sessions.data, data.Id)
			delete(h.sessions.chans, data.Id)
			h.sessions.RUnlock()
			break loop
		default:
		}
	}
}

func exit(ss *sessions) {
	// closing all sessions
	ss.RLock()
	for _, ch := range ss.chans {
		ch <- struct{}{}
	}
	for _, da := range ss.data {
		close(da)
	}
	ss.RUnlock()
}

type webConn struct {
	sync.Mutex
	conn *websocket.Conn
}

func writeResponse(w *webConn, typ, id string, r interface{}, er error) error {
	var payload []byte
	var err error
	if typ == "data" {
		if er != nil {
			payload, err = json.Marshal(httpResponse{Data: r, Errors: []*jerrors.Error{jerrors.ConvertError(er)}})
			if err != nil {
				return err
			}
		} else {
			payload, err = json.Marshal(httpResponse{Data: r, Errors: []*jerrors.Error{}})
			if err != nil {
				return err
			}
		}
	} else if typ == "error" || typ == "connection_error" {
		if er == nil {
			er = errors.New("connection is closed")
		}
		str := strings.Replace(er.Error(), "\"", "\\\"", -1)
		payload = json.RawMessage("{ \"error\" : \"" + str + "\"}")
	}
	res := wsMessage{
		Type:    typ,
		Id:      id,
		Payload: payload,
	}
	w.Lock()
	if err := w.conn.WriteJSON(res); err != nil {
		w.Unlock()
		return err
	}
	w.Unlock()
	return nil
}

func (h *httpSubHandler) serveHTTP(conn *webConn, data wsMessage, schema graphql.Type, query *graphql.Query, end chan struct{}, w http.ResponseWriter, r *http.Request) error {
	sid := data.Id
	sess := make(chan *event)
	h.sessions.Lock()
	h.sessions.data[sid] = sess
	h.sessions.chans[sid] = end
	h.sessions.Unlock()

	cls := func(ss *sessions, sid string) {
		ss.Lock()
		close(ss.data[sid])
		close(ss.chans[sid])
		ss.Unlock()
	}

	// Listening on usrChannel for any source event of subType
	for msg := range sess {
		select {
		case <-end:
			cls(h.sessions, sid)
			return nil
		default:
			// Subscription should have only one root selection
			// https://spec.graphql.org/June2018/#sec-Single-root-field
			if len(query.Selections) == 1 && query.Selections[0].Name == msg.typ {
				if err := func() error {
					res, err := h.executor.Execute(r.Context(), schema, &schemabuilder.Subscription{Payload: msg.payload}, query)
					if err == graphql.ErrNoUpdate {
						return nil
					}
					if err := writeResponse(conn, "data", data.Id, res, err); err != nil {
						return err
					}
					return nil
				}(); err != nil {
					cls(h.sessions, sid)
					return err
				}
			}
		}
	}
	return nil
}
