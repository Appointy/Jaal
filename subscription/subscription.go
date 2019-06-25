package subscription

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"sync"
)

type Publisher struct {}

func NewPublisher() *Publisher {
	return &Publisher{}
}

func (p *Publisher) WrapSourceEvent(name string, payload []byte) ([]byte, error) {
	var data bytes.Buffer
		evt := Event{
			Typ: name,
			Payload: payload,
		}
	if err := gob.NewEncoder(&data).Encode(evt); err != nil {
		return nil, err
	}
	return data.Bytes(), nil
}

type Event struct {
	Typ string
	Payload []byte
}

func (e *Event) GetType() string { return e.Typ }

func (e *Event) GetPayload() []byte { return e.Payload }

var Source = make(chan []byte)

// For each subscription type, a list of all connection channels to broadcast subscription type filtered source events
type typeNotif struct {
	Clients         map[string]chan []byte
	ServerTypeNotif chan chan []byte
}

type runtimeSubManager struct {
	ServerTypeNotifs map[string]*typeNotif
	Lock             *sync.RWMutex
}

type subTypeManager struct {
	SubTypeStreams map[string]chan []byte
}

// RuntimeSubManager stores all the connection streams for each subscription type
var RuntimeSubManager runtimeSubManager

// SubTypeManager stores all the subscription type and the source event streams
var SubTypeManager subTypeManager

func init() {
	RuntimeSubManager = runtimeSubManager{
		make(map[string]*typeNotif),
		&sync.RWMutex{},
	}
	SubTypeManager = subTypeManager{
		make(map[string]chan []byte),
	}
}

// RunSubscriptionServices launches all the daemons necessary for subscription implementation
func RunSubscriptionServices() {
	for k := range SubTypeManager.SubTypeStreams {
		go AddClientDaemon(k)
		go SourceSubTypeTrigger(k)
	}
	go SourceEventListener()
}

// RegisterSubType - Call at the server before/after making a subscription field func
func RegisterSubType(subType string) error {
	if _, ok := SubTypeManager.SubTypeStreams[subType]; ok {
		return fmt.Errorf("endpoint already registered")
	}
	SubTypeManager.SubTypeStreams[subType] = make(chan []byte)
	RuntimeSubManager.Lock.Lock()
	RuntimeSubManager.ServerTypeNotifs[subType] = &typeNotif{make(map[string]chan []byte), make(chan chan []byte, 1)}
	RuntimeSubManager.Lock.Unlock()
	fmt.Println("Registered", subType)
	return nil
}

// SourceEventListener listens for a source event and sends it to the corresponding subscription type stream
func SourceEventListener() {
	for e := range Source {
		fmt.Println("Got a source")
		var evt Event
		if err := gob.NewDecoder(bytes.NewReader(e)).Decode(&evt); err != nil {
			panic("failed to decode subscription data")
		}
		if _, ok := SubTypeManager.SubTypeStreams[evt.Typ]; !ok {
			panic("invalid sub type in source event")
		}
		SubTypeManager.SubTypeStreams[evt.Typ] <- evt.Payload
	}
}

// AddClientDaemon - Launch as go routine for every subscription type registered
func AddClientDaemon(subType string) {
	RuntimeSubManager.Lock.RLock()
	serverListener := RuntimeSubManager.ServerTypeNotifs[subType].ServerTypeNotif
	RuntimeSubManager.Lock.RUnlock()
	for client := range serverListener {
		// Add the client notifier in the server's log of clients for a particular subtype
		id := <-client
		fmt.Println("Received:", id)
		RuntimeSubManager.Lock.Lock()
		RuntimeSubManager.ServerTypeNotifs[subType].Clients[string(id)] = client
		RuntimeSubManager.Lock.Unlock()
		fmt.Println("stored client")
	}
}

// SourceSubTypeTrigger - Launch as go routine for every subscription type to listen for filtered source events from SubTypeStreams
func SourceSubTypeTrigger(subType string) {
	for i := range SubTypeManager.SubTypeStreams[subType] {
		fmt.Println("Received from stream")
		RuntimeSubManager.Lock.RLock()
		for k, v := range RuntimeSubManager.ServerTypeNotifs[subType].Clients {
			fmt.Println("Sending to client", k, "...")
			v <- i
			fmt.Println("Sent to client")
		}
		RuntimeSubManager.Lock.RUnlock()
	}
}

// Delete the connection channel from storage
func DeleteEntries(id string, subType string) {
	RuntimeSubManager.Lock.Lock()
	delete(RuntimeSubManager.ServerTypeNotifs[subType].Clients, id)
	RuntimeSubManager.Lock.Unlock()
}
