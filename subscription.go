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

type subTypeCacheManager struct {
	SubTypeCache map[string]interface{}
	CacheRead    map[string]int64
	Lock         *sync.Mutex
}

// SubStreamManager manages all the client streams and the source event streams for each subscription type
var SubStreamManager subStreamManager

// SubTypeCacheManager stores the source event for each subscription type to execute at clients
var SubTypeCacheManager subTypeCacheManager

func init() {
	SubStreamManager = subStreamManager{
		make(map[string]chan interface{}),
		make(map[string]*typeNotif),
		&sync.RWMutex{},
	}
	SubTypeCacheManager = subTypeCacheManager{
		make(map[string]interface{}),
		make(map[string]int64),
		&sync.Mutex{},
	}
}

// Call at the server before making a subscription field func
func registerSubTypeBase(subType string) error {
	SubStreamManager.Lock.RLock()
	if _, ok := SubStreamManager.SubTypeStreams[subType]; ok {
		SubStreamManager.Lock.RUnlock()
		return fmt.Errorf("Type already registered")
	}
	SubStreamManager.Lock.RUnlock()

	SubStreamManager.Lock.Lock()
	SubStreamManager.SubTypeStreams[subType] = make(chan interface{})
	SubStreamManager.ServerTypeNotifs[subType] = &typeNotif{make(map[string]chan interface{}, 0), make(chan chan interface{}, 1)}
	SubStreamManager.Lock.Unlock()
	return nil
}

// Launch as go routine for every subscription type registered
func addClientDaemon(subType string) {
	SubStreamManager.Lock.RLock()
	serverListener := SubStreamManager.ServerTypeNotifs[subType].ServerTypeNotif
	SubStreamManager.Lock.RUnlock()
	for client := range serverListener {
		// Add the client notifier in the server's log of clients for a particular subtype
		id := <-client
		SubStreamManager.Lock.Lock()
		SubStreamManager.ServerTypeNotifs[subType].Clients[id.(string)] = client
		SubStreamManager.Lock.Unlock()
	}
}

// Launch as go routine for every subscription type to listen for source events from SubTypeStreams
func sourceSubTypeTrigger(subType string) {
	SubStreamManager.Lock.RLock()
	subTypeListener := SubStreamManager.SubTypeStreams[subType]
	SubStreamManager.Lock.RUnlock()
	for i := range subTypeListener {
		// TODO : Update SubType cache
		SubStreamManager.Lock.RLock()
		for _, v := range SubStreamManager.ServerTypeNotifs[subType].Clients {
			v <- 1
		}
		SubStreamManager.Lock.RUnlock()
		SubTypeCacheManager.Lock.Lock()
		SubTypeCacheManager.SubTypeCache[subType] = i
		for {
			SubStreamManager.Lock.RLock()
			num := int64(len(SubStreamManager.ServerTypeNotifs[subType].Clients))
			SubStreamManager.Lock.RUnlock()
			SubTypeCacheManager.Lock.Lock()
			if num == SubTypeCacheManager.CacheRead[subType] {
				SubTypeCacheManager.Lock.Unlock()
				break
			}
			SubTypeCacheManager.Lock.Unlock()
		}
	}
}

func deleteEntries(id string, subType string) {
	SubStreamManager.Lock.Lock()
	delete(SubStreamManager.ServerTypeNotifs[subType].Clients, id)
	SubStreamManager.Lock.Unlock()
}
