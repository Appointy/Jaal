package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/gorilla/websocket"
)

type httpResponse struct {
	Data   interface{} `json:"data"`
	Errors []string    `json:"errors"`
}

type httpPostBody struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables"`
}

func main() {
	queryString := `subscription{
			channelStream(in: {name: "Table Saheb"}) {
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
		// var res httpResponse
		// if err := c.ReadJSON(&res); err != nil {
		// 	fmt.Println("Server disconnected: ", err)
		// 	return
		// }
		// fmt.Println(res)
		_, msg, err := c.ReadMessage()
		if err != nil {
			fmt.Println("Server disconnected:", err)
			return
		}
		fmt.Println(string(msg))
	}
}
