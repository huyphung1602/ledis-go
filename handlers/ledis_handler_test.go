package handlers_test

import (
	"net/http/httptest"
	"testing"

	"github.com/parnurzeal/gorequest"
	"github.com/zealotnt/ledis-go/handlers"
)

var serverUrl string

func SendCommand(cmd string) string {
	_, body, errs := gorequest.New().Post(serverUrl).Type("text").SendString(cmd).End()
	if errs != nil {
		panic(errs)
	}
	return body
}

func TestSetGet(t *testing.T) {
	handlers.InitStore()
	handler := &handlers.LedisHandler{}
	server := httptest.NewServer(handler)
	defer server.Close()
	serverUrl = server.URL

	body := SendCommand(`SET testkey 123`)
	expected := "OK"
	if expected != body {
		t.Errorf("Expected the message '%s', get '%s'\n", expected, body)
	}
	body = SendCommand(`GET testkey`)
	expected = "123"
	if expected != body {
		t.Errorf("Expected the message '%s', get '%s'\n", expected, body)
	}
}
