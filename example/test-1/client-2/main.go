package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"

	"github.com/gorilla/websocket"
)

type httpPostBody struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables"`
}

func main() {
	// r := bufio.NewReader(os.Stdin)
	// fmt.Println("Enter the tag you want to subscribe to - ")
	// name, err := r.ReadString('\n')
	// if err != nil {
	// 	fmt.Println("couldn't process user input")
	//	return
	// }
	queryString := `subscription{
			postStream(in: {tag: "Huer"}) {
				id
				tag
				title
			}
		}`

	reqQuery := httpPostBody{Query: queryString}
	query, err := json.Marshal(reqQuery)
	if err != nil {
		fmt.Println(err)
		return
	}
	header := make(http.Header)
	header.Add("query", string(query))
	u := url.URL{Scheme: "ws", Host: "localhost:3000", Path: "/graphql"}
	c, resp, err := websocket.DefaultDialer.Dial(u.String(), header)
	if err != nil {
		b, _ := ioutil.ReadAll(resp.Body)
		fmt.Println(string(b))
		fmt.Println("Could not connect to server: ", err)
		return
	}
	defer c.Close()

	for {
		if rand.Intn(100) < 20 {
			c.WriteMessage(1, []byte(""))
			continue
		}
		_, msg, err := c.ReadMessage()
		if err != nil {
			fmt.Println("Server disconnected:", err)
			return
		}
		fmt.Println(string(msg))
	}
}
