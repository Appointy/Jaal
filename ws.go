package jaal

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/appointy/idgen"
	"github.com/gorilla/websocket"
	"go.appointy.com/jaal/graphql"
	"go.appointy.com/jaal/schemabuilder"
	"go.appointy.com/jaal/subscription"
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
	qmHandler http.Handler
}

type endMessage struct{}

var upgrader = websocket.Upgrader{}

func (h *httpSubHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Println("started")
	//s, _ := ioutil.ReadAll(r.Body)
	//fmt.Println(r.URL.Query())
	//fmt.Println(string(s))
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

	//fmt.Println("Query: ", r.URL.Query().Get("query"))
	//fmt.Println("Variables: ", r.URL.Query().Get("variables"))

	//if r.Header.Get("query") == "" {
	//	res := getResponse(nil, errors.New("request must include a query"))
	//	w.Write(res)
	//	return
	//}
	q := r.URL.Query().Get("query")

	var params map[string]interface{}

	if err := json.NewDecoder(strings.NewReader(strings.Trim(r.URL.Query().Get("variables"), "\""))).Decode(&params); err != nil {
		res := getResponse(nil, err)
		w.Write(res)
		fmt.Println("failed to gob:", err)
		return
	}
	fmt.Println("parsed")
	query, err := graphql.Parse(q, params)
	if err != nil {
		res := getResponse(nil, err)
		w.Write(res)
		return
	}
	if query.Kind != "Subscription" {
		h.qmHandler.ServeHTTP(w, r)
		return
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
		res := getResponse(nil, errors.New("could not establish websocket connection"))
		fmt.Println(err)
		w.Write(res)
		return
	}
	defer conn.Close()

	id := idgen.New("usr")
	fmt.Println(id)

	usrChannel := make(chan []byte)
	subscription.RuntimeSubManager.Lock.RLock()
	subscription.RuntimeSubManager.ServerTypeNotifs[subType].ServerTypeNotif <- usrChannel
	subscription.RuntimeSubManager.Lock.RUnlock()
	usrChannel <- []byte(id)

	// Client side unsubscribe/disconnect signal
	var ext = make(chan int)

	// Check for unsubscription
	go func() {
		_, _, _ = conn.ReadMessage()
		ext <- 1
		return
	}()

	// For an extra loop so that the server doesn't block
	disconnect := false
	// Listening on usrChannel for any source event of subType
	for msg := range usrChannel {
		if disconnect {
			break
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
	subscription.DeleteEntries(id, subType)
	fmt.Printf("Client %v disconnected\n", id)
}
