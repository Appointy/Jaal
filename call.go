package jaal

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"

	"go.appointy.com/jaal/jerrors"
)

// HttpCall sends an HTTP Request to the specified url and returns response in map of map
func HttpCall(url, query string, variables map[string]interface{}, headers map[string]string) (map[string]interface{}, []*jerrors.Error) {
	var (
		requestBody = httpPostBody{
			Query:     query,
			Variables: variables,
		}
		responseBody httpResponse
	)

	client := http.Client{
		Timeout: time.Duration(500 * time.Second),
	}

	requestData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, []*jerrors.Error{jerrors.ConvertError(err)}
	}

	request, err := http.NewRequest("POST", url, bytes.NewBuffer(requestData))
	if err != nil {
		return nil, []*jerrors.Error{jerrors.ConvertError(err)}
	}

	for key, value := range headers {
		request.Header.Set(key, value)
	}

	response, err := client.Do(request)
	if err != nil {
		return nil, []*jerrors.Error{jerrors.ConvertError(err)}
	}

	responseData, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, []*jerrors.Error{jerrors.ConvertError(err)}
	}

	if err := json.Unmarshal(responseData, &responseBody); err != nil {
		return nil, []*jerrors.Error{jerrors.ConvertError(err)}
	}

	if len(responseBody.Errors) > 0 {
		return nil, responseBody.Errors
	}

	data, ok := (responseBody.Data).(map[string]interface{})
	if !ok {
		return nil, nil
	}

	return data, responseBody.Errors
}
