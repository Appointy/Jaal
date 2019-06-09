package jaal

import (
	"fmt"
	"sync"

	"github.com/gorilla/websocket"
	"go.appointy.com/jaal/graphql"
)

// SubTypeStreams contains the channels for each subscription type
// var SubTypeStreams = make(map[string](chan interface{}))

// User is a user subscription format
type User struct {
	conn     *websocket.Conn
	query    *graphql.Query
	response *(chan interface{})
}

// SubscriptionTypeManager stores all user subscriptions for a particular type - connections and subscription queries
type SubscriptionTypeManager struct {
	Users map[string]User
	lock  *sync.RWMutex
}

type SourceEvent struct{}

// type subscriptionManager map[string]SubscriptionTypeManager

// SubscriptionManager stores all the SubsrictionTypeManagers
var SubscriptionManager = make(map[string]SubscriptionTypeManager)

func registerSubType(subType string) error {
	// if _, ok := SubTypeStreams[name]; ok {
	// 	return fmt.Errorf("Type already registered")
	// }
	if _, ok := SubscriptionManager[subType]; ok {
		return fmt.Errorf("Type already registered")
	}
	SubscriptionManager[subType] = SubscriptionTypeManager{
		Users: make(map[string]User),
		lock:  &sync.RWMutex{},
	}
	// SubTypeStreams[subType] = make(chan interface{})
	return nil
}

// FilterSubs filters messages in subscription type and push to user's channels
func FilterSubs(subType string, responseEvent interface{}) {

	// responseEvent := <-SubTypeStreams[subType]

	for _, v := range SubscriptionManager[subType].Users {
		// TODO : Filter the messages according to the arguments
		SubscriptionManager[subType].lock.RLock()
		*(v.response) <- responseEvent
		SubscriptionManager[subType].lock.RUnlock()
	}
}

// MapSubType sorts incoming source events to the subscription type channels
func MapSubType(sourceEvent SourceEvent) {

	// SubTypeStreams[sourceEvent.(SourceResponse).subType] <- sourceEvent
}
