package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"sync"

	shellquote "github.com/kballard/go-shellquote"
)

var store *ledisStore

func main() {
	log.Printf("Ledis server started\n")
	addr := ":8080"

	store = &ledisStore{
		data: make(map[string]ledisData),
		lock: &sync.RWMutex{},
	}

	http.HandleFunc("/", ledisHandle)

	log.Printf("Accepting connections at: %s", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		panic(err)
	}
}

type ledisType int

const (
	TypeSet ledisType = iota
	TypeList
	TypeString
)

type ledisData struct {
	dataType   ledisType
	setData    *bool
	listData   *[]string
	stringData *string
}

type ledisStore struct {
	data map[string]ledisData
	lock *sync.RWMutex
}

func ledisHandle(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	body, _ := ioutil.ReadAll(r.Body)
	bodyStr := string(body[:])

	cmd, err := parseCommand(bodyStr)
	if err != nil {
		log.Printf("body: %v", err)
	}

	switch strings.ToUpper(cmd.Name) {
	case "GET":
		if len(cmd.Args) != 1 {
			respError(w, fmt.Errorf("GET expects 1 argument"))
			return
		}
		val := store.Get(cmd.Args[0])
		writeBody(w, val)
	case "SET":
		if len(cmd.Args) != 2 {
			respError(w, fmt.Errorf("SET expects 2 arguments"))
			return
		}
		store.Set(cmd.Args[0], cmd.Args[1])
		writeBody(w, "OK")
		return
	default:
		respError(w, fmt.Errorf("unkonwn command: %s", cmd.Name))
	}
}

type command struct {
	Name string
	Args []string
}

func writeBody(w http.ResponseWriter, body string) {
	io.WriteString(w, body)
}

func respError(w http.ResponseWriter, err error) {
	writeBody(w, fmt.Sprintf("ERROR: %s", err.Error()))
}

func parseCommand(body string) (*command, error) {
	args, err := shellquote.Split(body)
	if err != nil {
		return nil, err
	}

	return &command{Name: args[0], Args: args[1:]}, nil
}

func (store *ledisStore) Get(key string) string {
	store.lock.RLock()
	defer store.lock.RUnlock()

	if store.data[key].stringData == nil {
		return "(nil)"
	}
	return *store.data[key].stringData
}

func (store *ledisStore) Set(key string, val string) {
	store.lock.Lock()
	defer store.lock.Unlock()

	store.data[key] = ledisData{
		dataType:   TypeString,
		setData:    nil,
		listData:   nil,
		stringData: &val}
}
