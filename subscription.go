package jaal

import (
	"context"
	"fmt"
	"reflect"
	"sync"

	"cloud.google.com/go/pubsub"
)

/* TODO :
SEPARATE SUBTYPESTREAMS BECAUSE THEY DON'T NEED LOCKS.
MAKE DIFFERENT STRUCTS FOR LOCK AND NON-LOCK IMPLEMENTED VARIABLES
*/

type Subscription interface {
	Receive(context.Context, func(context.Context, *pubsub.Message)) error
}

// for each subscription type list of all user channels to pass them source events
type typeNotif struct {
	Clients         map[string]chan interface{}
	ServerTypeNotif chan chan interface{}
}

type subStreamManager struct {
	SubTypeStreams   map[string]chan interface{}
	ServerTypeNotifs map[string]*typeNotif
	Lock             *sync.RWMutex
}

// SubStreamManager manages all the client streams and the source event streams for each subscription type
var SubStreamManager subStreamManager

// SourceStream ...
var SourceStream = make(chan interface{})

var resolvers = make(map[string]interface{})

func init() {
	SubStreamManager = subStreamManager{
		make(map[string]chan interface{}),
		make(map[string]*typeNotif),
		&sync.RWMutex{},
	}
}

// RegisterSubType - RCall at the server before making a subscription field func
func RegisterSubType(subType string, resolver interface{}) error {

	if err := checkResolver(resolver); err != nil {
		return err
	}

	SubStreamManager.Lock.RLock()
	if _, ok := SubStreamManager.SubTypeStreams[subType]; ok {
		SubStreamManager.Lock.RUnlock()
		return fmt.Errorf("Type already registered")
	}
	SubStreamManager.Lock.RUnlock()

	resolvers[subType] = resolver

	SubStreamManager.Lock.Lock()
	SubStreamManager.SubTypeStreams[subType] = make(chan interface{}, 1)
	SubStreamManager.ServerTypeNotifs[subType] = &typeNotif{make(map[string]chan interface{}), make(chan chan interface{}, 1)}
	SubStreamManager.Lock.Unlock()
	return nil
}

// Source event to subscription type event resolvers should be of type - func(anyType) anyType, error
func checkResolver(f interface{}) error {
	v := reflect.ValueOf(f)
	if v.Kind() != reflect.Func {
		return fmt.Errorf("Resolver should be of type func(anyType) (anyType, error)")
	}
	if v.Type().NumOut() != 2 {
		return fmt.Errorf("Resolver should have only 2 return values")
	}
	errorType := reflect.TypeOf((*error)(nil)).Elem()
	if v.Type().Out(1) != errorType {
		return fmt.Errorf("Resolver's second return type should be error type")
	}
	return nil
}

//SourceEventListener ...
func SourceEventListener(ctx context.Context, sub Subscription) {
	//-------------------For a Google PubSub subscription type----------------------------
	if err := sub.Receive(ctx, func(ctx context.Context, m *pubsub.Message) {
		for v, k := range resolvers {
			f := reflect.ValueOf(k)
			fmt.Println("Calling resolver")
			output := f.Call([]reflect.Value{reflect.ValueOf(m)})
			if output[1].Interface() != nil {
				continue
			}
			fmt.Println("Sent into", v, "stream")
			SubStreamManager.Lock.RLock()
			SubStreamManager.SubTypeStreams[v] <- output[0].Interface()
			SubStreamManager.Lock.RUnlock()
		}
		m.Ack()
	}); err != nil {
		fmt.Println("Error in recieving PubSub message")
	}
}

// AddClientDaemon - Launch as go routine for every subscription type registered
func AddClientDaemon(subType string) {
	SubStreamManager.Lock.RLock()
	serverListener := SubStreamManager.ServerTypeNotifs[subType].ServerTypeNotif
	SubStreamManager.Lock.RUnlock()
	for client := range serverListener {
		// Add the client notifier in the server's log of clients for a particular subtype
		id := <-client
		fmt.Println("Received:", id)
		SubStreamManager.Lock.Lock()
		SubStreamManager.ServerTypeNotifs[subType].Clients[id.(string)] = client
		SubStreamManager.Lock.Unlock()
		fmt.Println("stored client")

	}
}

// SourceSubTypeTrigger - Launch as go routine for every subscription type to listen for source events from SubTypeStreams
func SourceSubTypeTrigger(subType string) {
	SubStreamManager.Lock.RLock()
	subTypeListener := SubStreamManager.SubTypeStreams[subType]
	SubStreamManager.Lock.RUnlock()
	for i := range subTypeListener {
		fmt.Println("Received from stream")
		SubStreamManager.Lock.RLock()
		for k, v := range SubStreamManager.ServerTypeNotifs[subType].Clients {
			fmt.Println("Sending to client", k, "...")
			v <- i
			fmt.Println("Sent to client")
		}
		SubStreamManager.Lock.RUnlock()
	}
}

func deleteEntries(id string, subType string) {
	SubStreamManager.Lock.Lock()
	delete(SubStreamManager.ServerTypeNotifs[subType].Clients, id)
	SubStreamManager.Lock.Unlock()
}
