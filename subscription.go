package jaal

import (
	"fmt"
	"sync"
)

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

func init() {
	SubStreamManager = subStreamManager{
		make(map[string]chan interface{}),
		make(map[string]*typeNotif),
		&sync.RWMutex{},
	}
}

// RegisterSubType - RCall at the server before making a subscription field func
func RegisterSubType(subType string) error {
	SubStreamManager.Lock.RLock()
	if _, ok := SubStreamManager.SubTypeStreams[subType]; ok {
		SubStreamManager.Lock.RUnlock()
		return fmt.Errorf("Type already registered")
	}
	SubStreamManager.Lock.RUnlock()

	SubStreamManager.Lock.Lock()
	SubStreamManager.SubTypeStreams[subType] = make(chan interface{}, 1)
	SubStreamManager.ServerTypeNotifs[subType] = &typeNotif{make(map[string]chan interface{}), make(chan chan interface{}, 1)}
	SubStreamManager.Lock.Unlock()
	return nil
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
