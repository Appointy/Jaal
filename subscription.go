package jaal

import (
	"context"
	"fmt"
	"reflect"
	"sync"

	"cloud.google.com/go/pubsub"
)

// Subscription is an interface implemented for source stream subscriptions
type Subscription interface {
	Receive(context.Context, func(context.Context, *pubsub.Message)) error
}

// For each subscription type, a list of all connection channels to broadcast subscription type filtered source events
type typeNotif struct {
	Clients         map[string]chan interface{}
	ServerTypeNotif chan chan interface{}
}

type runtimeSubManager struct {
	ServerTypeNotifs map[string]*typeNotif
	Lock             *sync.RWMutex
}

type subTypeManager struct {
	SubTypeStreams map[string]chan interface{}
	SourceStream   chan interface{}
	Resolvers      map[string]interface{}
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
		make(map[string]chan interface{}),
		make(chan interface{}),
		make(map[string]interface{}),
	}
}

// RunSubscriptionServices launches all the daemons necessary for subscription implementation
func RunSubscriptionServices(ctx context.Context, sub Subscription) {
	for k := range SubTypeManager.SubTypeStreams {
		go AddClientDaemon(k)
		go SourceSubTypeTrigger(k)
	}
	go SourceEventListener(ctx, sub)
}

// RegisterSubType - Call at the server before/after making a subscription field func
func RegisterSubType(subType string, resolver interface{}) error {

	if err := checkResolver(resolver); err != nil {
		return err
	}
	if _, ok := SubTypeManager.SubTypeStreams[subType]; ok {
		return fmt.Errorf("Type already registered")
	}

	SubTypeManager.Resolvers[subType] = resolver
	SubTypeManager.SubTypeStreams[subType] = make(chan interface{}, 1)
	RuntimeSubManager.Lock.Lock()
	RuntimeSubManager.ServerTypeNotifs[subType] = &typeNotif{make(map[string]chan interface{}), make(chan chan interface{}, 1)}
	RuntimeSubManager.Lock.Unlock()
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

// SourceEventListener listens for a source event and sends it to the corresponding subscription type stream
func SourceEventListener(ctx context.Context, sub Subscription) {
	//-------------------For a Google PubSub subscription type----------------------------
	if err := sub.Receive(ctx, func(ctx context.Context, m *pubsub.Message) {
		for v, k := range SubTypeManager.Resolvers {
			f := reflect.ValueOf(k)
			fmt.Println("Calling resolver")
			output := f.Call([]reflect.Value{reflect.ValueOf(m)})
			if output[1].Interface() != nil {
				continue
			}
			fmt.Println("Sent into", v, "stream")
			SubTypeManager.SubTypeStreams[v] <- output[0].Interface()
		}
		m.Ack()
	}); err != nil {
		fmt.Println("Error in receiving PubSub message")
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
		RuntimeSubManager.ServerTypeNotifs[subType].Clients[id.(string)] = client
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
func deleteEntries(id string, subType string) {
	RuntimeSubManager.Lock.Lock()
	delete(RuntimeSubManager.ServerTypeNotifs[subType].Clients, id)
	RuntimeSubManager.Lock.Unlock()
}
