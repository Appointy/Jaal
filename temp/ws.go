package temp

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"sync"

	"github.com/appointy/idgen"
	"github.com/gorilla/websocket"
	"go.appointy.com/jaal/graphql"
	"go.appointy.com/jaal/schemabuilder"
)

// HTTPSubHandler implements the handler required for executing the graphql subscriptions
func HTTPSubHandler(schema *graphql.Schema) http.Handler {
	return &httpSubHandler{
		handler{
			schema:   schema,
			executor: &graphql.Executor{},
		},
		HTTPHandler(schema),
	}
}

type httpSubHandler struct {
	handler
	hand http.Handler
}

type endMessage struct{}

var upgrader = websocket.Upgrader{}

var getResponse = func(value interface{}, err error) []byte {
	response := httpResponse{}
	if err != nil {
		response.Errors = []string{err.Error()}
	} else {
		response.Data = value
	}

	responseJSON, err := json.Marshal(response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return nil
	}
	return responseJSON
}

func (h *httpSubHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Println(r.URL)

	fmt.Println("started")

	if r.Header.Get("query") == "" {
		res := getResponse(nil, errors.New("request must include a query"))
		w.Write(res)
		return
	}

	var params httpPostBody
	if err := json.NewDecoder(strings.NewReader(r.Header.Get("query"))).Decode(&params); err != nil {
		res := getResponse(nil, err)
		w.Write(res)
		return
	}

	query, err := graphql.Parse(params.Query, params.Variables)
	if err != nil {
		res := getResponse(nil, err)
		w.Write(res)
		return
	}
	fmt.Println("parsed fucking boi!")
	if query.Kind != "subscription" {
		h.hand.ServeHTTP(w, r)
	}

	subType := query.SelectionSet.Selections[0].Name

	fmt.Println("parsed, subType:", subType)

	schema := h.schema.Subscription

	if err := graphql.ValidateQuery(r.Context(), schema, query.SelectionSet); err != nil {
		res := getResponse(nil, err)
		w.Write(res)
		return
	}

	fmt.Println("validated")

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		res := getResponse(nil, fmt.Errorf("could not establish websokcet connection: %v", err))
		fmt.Println(err)
		w.Write(res)
		return
	}
	defer conn.Close()

	id := idgen.New("usr")
	fmt.Println(id)
	// Refers this websocket instance to the server
	usrChannel := make(chan interface{})
	RuntimeSubManager.Lock.RLock()
	RuntimeSubManager.ServerTypeNotifs[subType].ServerTypeNotif <- usrChannel
	RuntimeSubManager.Lock.RUnlock()
	usrChannel <- id

	v := make(chan interface{})
	userSubs := &userChannels{
		subs : []chan interface{}{v},
		lock : &sync.Mutex{},
	}
	go SubConnection(v, subType, conn, schema, query)
	// Check for disconnection or any other queries
	for {
		_, req, err := conn.ReadMessage()
		if err != nil {
			userSubs.lock.Lock()
			for _, v := range userSubs.subs {
				v <- end{}
			}
			userSubs.lock.Unlock()
			return
		}
		// TODO : Parse, validate and execute according to type, preferably by separating it into a function
		subType := ""
		// After approval of the subscription query
		v := make(chan interface{})
		userSubs.lock.Lock()
		userSubs.subs = append(userSubs.subs, v)
		userSubs.lock.Unlock()
		go SubConnection(v, subType)
	}
}

type userChannels struct {
	subs []chan interface{}
	lock *sync.Mutex
}

type end struct{}

func SubConnection(channel chan interface{}, subType string, conn *websocket.Conn, schema graphql.Type, query string) {
	// For an extra loop so that the server doesn't block
	disconnect := false
	// Listening on usrChannel for any source event of subType
	for msg := range channel {
		if disconnect {
			break
		}
		if _, ok := msg.(end); ok {
			disconnect = true
			continue
		}
		fmt.Println("Received from server")
		select {
		case <-ext:
			disconnect = true
		default:
			output, err := h.executor.Execute(r.Context(), schema, &schemabuilder.Subscription{msg}, query)
			if err != nil {
				res := getResponse(nil, err)
				conn.WriteJSON(res)
				disconnect = true
				fmt.Println(err)
			}
			// In case of pointer return type for subscription type resolver, filter out the null reponses
			if reflect.TypeOf(output.(map[string]interface{})[subType]) != nil {
				conn.WriteMessage(1, getResponse(output, nil))
			}

		}
	}
	deleteEntries(id, subType)
	fmt.Printf("Client %v disconnected\n", id)
}