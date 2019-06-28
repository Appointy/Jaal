package jaal

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
	"gocloud.dev/pubsub"

	"go.appointy.com/jaal/graphql"
	"go.appointy.com/jaal/schemabuilder"
)

// HTTPSubHandler implements the handler required for executing the graphql subscriptions
func HTTPSubHandler(schema *graphql.Schema, s *pubsub.Subscription) (http.Handler, func()) {
	source := make(chan *event)
	sessions := &sessions{
		data:  map[string]chan *event{},
		chans: map[string]chan struct{}{},
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
			sessions:  sessions,
			done:      done,
		}, func() {
			go startListening(s, source, func() {
				close(done)
			})
			go listenSource(source, sessions)
		}
}

func listenSource(events chan *event, ss *sessions) {
	for evt := range events {
		ss.RLock()
		for _, s := range ss.data {
			s <- evt
		}
		ss.RUnlock()
	}
}

func startListening(s *pubsub.Subscription, source chan<- *event, cancel func()) {
	for {
		msg, err := s.Receive(context.Background())
		if err != nil {
			fmt.Println("Pubsub failed: ", err)
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
	done      <-chan struct{}
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
	// Check origin and set response headers
	h.upgrader.CheckOrigin = func(r *http.Request) bool { return true }
	res := http.Header{}
	res["Sec-Websocket-Protocol"] = []string{"graphql-ws"}

	conn, err := h.upgrader.Upgrade(w, r, res)
	if err != nil {
		fmt.Println("failed to upgrade to websocket:", err)
		return
	}
	defer conn.Close()

	if conn.Subprotocol() != "graphql-ws" {
		fmt.Println("invalid subprotocol")
		return
	}

	var msg wsMessage

	if err := conn.ReadJSON(&msg); err != nil {
		fmt.Println("failed to parse websocket message: ", err)
		return
	}
	wr, err := conn.NextWriter(websocket.TextMessage)
	if err != nil {
		fmt.Println("failed to get websocket writer:", err)
		return
	}
	defer wr.Close()
	if msg.Type == "connection_init" {
		// TODO : Verify and apply connectionParams
		if false {
			if err := writeResponse(wr, "connection_error", "", nil, fmt.Errorf("an error occured")); err != nil {
				fmt.Println(err)
				return
			}
		}
		if err := writeResponse(wr, "connection_ack", "", nil, nil); err != nil {
			fmt.Println(err)
			return
		}

	loop:
		for {
			select {
			case <-h.done:
				exit(h.sessions)
				return
			default:
				var data wsMessage
				if err := conn.ReadJSON(&data); err != nil {
					fmt.Println(err)
				}
				wr, err := conn.NextWriter(websocket.TextMessage)
				if err != nil {
					fmt.Println(err)
				}
				switch data.Type {
				case "start":
					end := make(chan struct{}, 1)
					go func(conn *websocket.Conn, data *wsMessage, end chan struct{}, w http.ResponseWriter, r *http.Request) {
						if err := h.serveHTTP(conn, *data, end, w, r); err != nil {
							fmt.Println("Id:", data.Id, ": terminated: ", err)
						}
						wr, err = conn.NextWriter(websocket.TextMessage)
						if err != nil {
							fmt.Println(err)
							return
						}
						defer wr.Close()
						if err := writeResponse(wr, "complete", data.Id, nil, nil); err != nil {
							fmt.Println(err)
						}
					}(conn, &data, end, w, r)
				case "stop":
					h.sessions.RLock()
					h.sessions.chans[data.Id] <- struct{}{}
					h.sessions.RUnlock()
				case "connection_terminate":
					exit(h.sessions)
					break loop
				default:
				}
			}
		}
	}
}

func exit(ss *sessions) {
	ss.RLock()
	for _, v := range ss.chans {
		v <- struct{}{}
	}
	ss.RUnlock()
}

func writeResponse(w io.Writer, typ, id string, r interface{}, er error) error {
	var payload []byte
	var err error
	if typ == "data" {
		if er != nil {
			payload, err = json.Marshal(httpResponse{Data: r, Errors: []string{er.Error()}})
			if err != nil {
				return err
			}
		} else {
			payload, err = json.Marshal(httpResponse{Data: r, Errors: []string{}})
			if err != nil {
				return err
			}
		}
	} else if typ == "error" || typ == "connection_error" {
		str := strings.Replace(er.Error(), "\"", "\\\"", -1)
		payload = json.RawMessage("{ \"error\" : \"" + str + "\"}")
	}
	res := wsMessage{
		Type:    typ,
		Id:      id,
		Payload: payload,
	}
	return json.NewEncoder(w).Encode(&res)
}

func (h *httpSubHandler) serveHTTP(conn *websocket.Conn, data wsMessage, end chan struct{}, w http.ResponseWriter, r *http.Request) error {
	var gql gqlPayload
	if err := json.Unmarshal(data.Payload, &gql); err != nil {
		return err
	}
	wr, err := conn.NextWriter(websocket.TextMessage)
	if err != nil {
		return err
	}
	query, err := graphql.Parse(gql.Query, gql.Variables)
	if err != nil {
		if er := writeResponse(wr, "error", data.Id, nil, err); er != nil {
			return er
		}
		return err
	}

	if len(query.SelectionSet.Selections) != 1 {
		return fmt.Errorf("exactly one subscription is expected")
	}

	schema := h.schema.Subscription
	if err := graphql.ValidateQuery(r.Context(), schema, query.SelectionSet); err != nil {
		if er := writeResponse(wr, "error", data.Id, nil, err); er != nil {
			return er
		}
		return err
	}

	sid := data.Id
	sess := make(chan *event)
	h.sessions.Lock()
	h.sessions.data[sid] = sess
	h.sessions.chans[sid] = end
	h.sessions.Unlock()

	cls := func(ss *sessions, sid string) {
		ss.Lock()
		s := ss.data[sid]
		e := ss.chans[sid]
		delete(ss.data, sid)
		delete(ss.data, sid)
		ss.Unlock()
		close(s)
		close(e)
		for range s {
		}
	}

	// Listening on usrChannel for any source event of subType
	for msg := range sess {
		select {
		case <-end:
			cls(h.sessions, sid)
			return nil
		default:
			if err := func() error {
				res, err := h.executor.Execute(r.Context(), schema, &schemabuilder.Subscription{msg.payload}, query)
				if err == graphql.ErrNoUpdate {
					return nil
				}
				rer := err
				if err := writeResponse(wr, "data", data.Id, res, rer); err != nil {
					return err
				}
				wr, err = conn.NextWriter(websocket.TextMessage)
				if err != nil {
					return err
				}
				if rer != nil {
					return err
				}
				return nil
			}(); err != nil {
				cls(h.sessions, sid)
				return err
			}
		}
	}
	return nil
}
