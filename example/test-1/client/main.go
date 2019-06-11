package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"

	"github.com/gorilla/websocket"
)

type httpPostBody struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables"`
}

func main() {
	r := bufio.NewReader(os.Stdin)
	fmt.Println("Enter the name you want to subscribe to - ")
	name, err := r.ReadString('\n')
	if err != nil {
		fmt.Println("couldn't process user input")
	}
	queryString := `subscription{
			channelStream(in: {name: "` + name[0:len(name)-1] + `"}) {
				id
				email
				name
			}
		}`

	reqQuery := httpPostBody{Query: queryString}
	query, err := json.Marshal(reqQuery)
	if err != nil {
		fmt.Println(err)
		return
	}
	header := make(http.Header)
	header.Add("body", string(query))
	u := url.URL{Scheme: "ws", Host: "localhost:3000", Path: "/graphql/sub"}
	c, resp, err := websocket.DefaultDialer.Dial(u.String(), header)
	if err != nil {
		b, _ := ioutil.ReadAll(resp.Body)
		fmt.Println(string(b))
		fmt.Println("Could not connect to server: ", err)
		return
	}
	defer c.Close()

	for {
		_, msg, err := c.ReadMessage()
		if err != nil {
			fmt.Println("Server disconnected:", err)
			return
		}
		fmt.Println(string(msg))
	}
}
