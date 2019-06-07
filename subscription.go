package jaal

import (
	"sync"

	"github.com/gorilla/websocket"
	"go.appointy.com/jaal/graphql"
)

// ALL SUBSCRIPTION MASTER INFO

var storeSub = struct {
	conn  map[string]*websocket.Conn
	query map[string]*graphql.Query
	lock  sync.RWMutex
}{
	conn:  make(map[string]*websocket.Conn),
	query: make(map[string]*graphql.Query),
}

func build() {

}
