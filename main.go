package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	shellquote "github.com/kballard/go-shellquote"
)

var store *ledisStore

func main() {
	log.Printf("Ledis server started\n")
	addr := ":8080"

	store = &ledisStore{
		data:   make(map[string]ledisData),
		expire: make(map[string]int64),
		lock:   &sync.RWMutex{},
	}

	go expiredCleaner()
	http.HandleFunc("/", ledisHandle)

	log.Printf("Accepting connections at: %s", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		panic(err)
	}
}

func expiredCleaner() {
	for {
		time.Sleep(500 * time.Millisecond)
		store.lock.RLock()

		timeNow := time.Now().Unix()
		for key, val := range store.expire {
			if val-timeNow <= 0 {
				delete(store.expire, key)
				delete(store.data, key)
			}
		}
		store.lock.RUnlock()
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
	setData    *map[string]bool
	listData   *[]string
	stringData *string
}

type ledisStore struct {
	data   map[string]ledisData
	expire map[string]int64
	lock   *sync.RWMutex
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
	case "LLEN":
		if len(cmd.Args) != 1 {
			respError(w, fmt.Errorf("LLEN expects 1 argument"))
			return
		}
		writeBody(w, store.Llen(cmd.Args[0]))
	case "RPUSH":
		if len(cmd.Args) <= 1 {
			respError(w, fmt.Errorf("RPUSH expects at least 2 arguments"))
			return
		}
		writeBody(w, store.Rpush(cmd.Args[0], cmd.Args[1:]))
	case "LPOP":
		if len(cmd.Args) != 1 {
			respError(w, fmt.Errorf("LPOP expects at least 2 arguments"))
			return
		}
		writeBody(w, store.Lpop(cmd.Args[0]))
	case "RPOP":
		if len(cmd.Args) != 1 {
			respError(w, fmt.Errorf("RPOP expects at least 2 arguments"))
			return
		}
		writeBody(w, store.Rpop(cmd.Args[0]))
	case "LRANGE":
		if len(cmd.Args) != 3 {
			respError(w, fmt.Errorf("RPOP expects 3 arguments"))
			return
		}
		startIdx, err := strconv.ParseUint(cmd.Args[1], 10, 64)
		if err != nil {
			respError(w, fmt.Errorf("Error when parsing start"))
			return
		}
		endIdx, err := strconv.ParseUint(cmd.Args[2], 10, 64)
		if err != nil {
			respError(w, fmt.Errorf("Error when parsing end"))
			return
		}
		writeBody(w, store.Lrange(cmd.Args[0], startIdx, endIdx))
	case "SADD":
		if len(cmd.Args) <= 1 {
			respError(w, fmt.Errorf("SADD expects at least 2 arguments"))
			return
		}
		writeBody(w, store.Sadd(cmd.Args[0], cmd.Args[1:]))
	case "SCARD":
		if len(cmd.Args) != 1 {
			respError(w, fmt.Errorf("SCARD expects 1 arguments"))
			return
		}
		writeBody(w, store.Scard(cmd.Args[0]))
	case "SMEMBERS":
		if len(cmd.Args) != 1 {
			respError(w, fmt.Errorf("SMEMBERS expects 1 arguments"))
			return
		}
		writeBody(w, store.Smembers(cmd.Args[0]))
	case "SREM":
		if len(cmd.Args) <= 1 {
			respError(w, fmt.Errorf("SREM expects at least 2 arguments"))
			return
		}
		writeBody(w, store.Srem(cmd.Args[0], cmd.Args[1:]))
	case "SINTER":
		if len(cmd.Args) <= 1 {
			respError(w, fmt.Errorf("SINTER expects at least 2 arguments"))
			return
		}
		writeBody(w, store.Sinter(cmd.Args))
	case "KEYS":
		writeBody(w, store.Keys())
	case "DEL":
		if len(cmd.Args) != 1 {
			respError(w, fmt.Errorf("DEL expects 1 argument"))
			return
		}
		writeBody(w, store.Del(cmd.Args[0]))
	case "FLUSHDB":
		writeBody(w, store.Flushdb())
	case "EXPIRE":
		if len(cmd.Args) != 2 {
			respError(w, fmt.Errorf("EXPIRE expects 2 arguments"))
			return
		}
		second, err := strconv.ParseInt(cmd.Args[1], 10, 64)
		if err != nil {
			respError(w, fmt.Errorf("Error when parsing end"))
			return
		}
		if second <= 0 {
			respError(w, fmt.Errorf("Second should be a positive number"))
			return
		}
		writeBody(w, store.Expire(cmd.Args[0], second))
	case "TTL":
		if len(cmd.Args) != 1 {
			respError(w, fmt.Errorf("TTL expects 1 argument"))
			return
		}
		writeBody(w, store.Ttl(cmd.Args[0]))
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

	// set always success, it even overwrite other data types
	store.data[key] = ledisData{
		dataType:   TypeString,
		setData:    nil,
		listData:   nil,
		stringData: &val}
	// remove the expire if exist
	delete(store.expire, key)
}

func (store *ledisStore) Llen(key string) string {
	store.lock.Lock()
	defer store.lock.Unlock()

	if store.data[key].dataType != TypeList {
		return "WRONGTYPE Operation against a key holding the wrong kind of value"
	}

	return fmt.Sprintf("%d", len(*store.data[key].listData))
}

func (store *ledisStore) Rpush(key string, values []string) string {
	store.lock.Lock()
	defer store.lock.Unlock()

	storeVal, ok := store.data[key]
	if ok {
		if storeVal.dataType != TypeList {
			return "WRONGTYPE Operation against a key holding the wrong kind of value"
		}

		// append value
		for _, val := range values {
			*store.data[key].listData = append(*store.data[key].listData, val)
		}
		return fmt.Sprintf("%d", len(*storeVal.listData))
	}

	// create the list
	store.data[key] = ledisData{
		dataType:   TypeList,
		setData:    nil,
		listData:   &values,
		stringData: nil}
	return fmt.Sprintf("%d", len(values))
}

func (store *ledisStore) Lpop(key string) string {
	store.lock.Lock()
	defer store.lock.Unlock()

	// check if key is exist
	storeVal, ok := store.data[key]
	if !ok {
		return "(nil)"
	}

	// if key is not list, return wrong type
	if storeVal.dataType != TypeList {
		return "WRONGTYPE Operation against a key holding the wrong kind of value"
	}

	// else, lpop
	if len(*storeVal.listData) == 0 {
		return "(nil)"
	}
	retVal := (*storeVal.listData)[0]
	*storeVal.listData = append((*storeVal.listData)[:0], (*storeVal.listData)[1:]...)
	return retVal
}

func (store *ledisStore) Rpop(key string) string {
	store.lock.Lock()
	defer store.lock.Unlock()

	// check if key is exist
	storeVal, ok := store.data[key]
	if !ok {
		return "(nil)"
	}

	// if key is not list, return wrong type
	if storeVal.dataType != TypeList {
		return "WRONGTYPE Operation against a key holding the wrong kind of value"
	}

	// else, rpop
	if len(*storeVal.listData) == 0 {
		return "(nil)"
	}
	lastIdx := len(*storeVal.listData) - 1
	retVal := (*storeVal.listData)[lastIdx]
	*storeVal.listData = append((*storeVal.listData)[:lastIdx], (*storeVal.listData)[lastIdx+1:]...)
	return retVal
}

func (store *ledisStore) Lrange(key string, start, stop uint64) string {
	store.lock.Lock()
	defer store.lock.Unlock()

	// check if key is exist
	storeVal, ok := store.data[key]
	if !ok {
		return "(nil)"
	}

	// if key is not list, return wrong type
	if storeVal.dataType != TypeList {
		return "WRONGTYPE Operation against a key holding the wrong kind of value"
	}

	lenListData := uint64(len(*storeVal.listData))
	if lenListData == 0 {
		return "(nil)"
	}

	stopIdx := stop
	if stopIdx >= lenListData {
		stopIdx = lenListData
	}

	retStr := ""
	for i := start; i < stopIdx; i++ {
		retStr += (*storeVal.listData)[i] + "\r\n"
	}
	return retStr
}

func (store *ledisStore) Sadd(key string, values []string) string {
	store.lock.Lock()
	defer store.lock.Unlock()

	count := 0
	storeVal, ok := store.data[key]
	if ok {
		if storeVal.dataType != TypeSet {
			return "WRONGTYPE Operation against a key holding the wrong kind of value"
		}

		// add item to set
		setVals := *storeVal.setData
		for _, val := range values {
			if _, ok = setVals[val]; !ok {
				count++
			}
			storeVal.setData = &setVals
		}
		return fmt.Sprintf("%d", count)
	}

	// not exist, create set
	setVals := make(map[string]bool)
	for _, val := range values {
		if _, ok := setVals[val]; !ok {
			count++
		}
		setVals[val] = true
	}
	store.data[key] = ledisData{
		dataType:   TypeSet,
		setData:    &setVals,
		listData:   nil,
		stringData: nil}
	return fmt.Sprintf("%d", len(values))
}

func (store *ledisStore) Scard(key string) string {
	store.lock.Lock()
	defer store.lock.Unlock()

	count := 0
	storeVal, ok := store.data[key]
	if !ok {
		return "0"
	}
	if storeVal.dataType != TypeSet {
		return "WRONGTYPE Operation against a key holding the wrong kind of value"
	}

	for _ = range *storeVal.setData {
		count++
	}
	return fmt.Sprintf("%d", count)
}

func (store *ledisStore) Smembers(key string) string {
	store.lock.Lock()
	defer store.lock.Unlock()

	resStr := ""
	storeVal, ok := store.data[key]
	if !ok {
		return "(empty set)"
	}
	if storeVal.dataType != TypeSet {
		return "WRONGTYPE Operation against a key holding the wrong kind of value"
	}

	if len(*storeVal.setData) == 0 {
		return "(empty set)"
	}

	for key := range *storeVal.setData {
		resStr += key + "\r\n"
	}
	return resStr
}

func (store *ledisStore) Srem(key string, values []string) string {
	store.lock.Lock()
	defer store.lock.Unlock()

	count := 0
	storeVal, ok := store.data[key]
	if !ok {
		return "0"
	}
	if storeVal.dataType != TypeSet {
		return "WRONGTYPE Operation against a key holding the wrong kind of value"
	}

	if len(*storeVal.setData) == 0 {
		return "0"
	}

	for _, val := range values {
		if _, ok := (*storeVal.setData)[val]; ok {
			count++
			delete(*storeVal.setData, val)
		}
	}

	return fmt.Sprintf("%d", count)
}

func (store *ledisStore) Sinter(key []string) string {
	store.lock.Lock()
	defer store.lock.Unlock()
	// TODO: implement Sinter
	return ""
}

func (store *ledisStore) Keys() string {
	store.lock.Lock()
	defer store.lock.Unlock()

	valRet := ""
	for key := range store.data {
		valRet += key + "\r\n"
	}

	if valRet == "" {
		return "empty"
	}
	return valRet
}

func (store *ledisStore) Del(key string) string {
	store.lock.Lock()
	defer store.lock.Unlock()

	if _, ok := store.data[key]; !ok {
		return "0"
	}

	delete(store.data, key)
	return "1"
}

func (store *ledisStore) Flushdb() string {
	store.lock.Lock()
	defer store.lock.Unlock()
	for key := range store.data {
		delete(store.data, key)
	}
	return "OK"
}

func (store *ledisStore) Expire(key string, second int64) string {
	store.lock.Lock()
	defer store.lock.Unlock()

	if _, ok := store.data[key]; !ok {
		return "key not found"
	}

	store.expire[key] = time.Now().Unix() + second
	return fmt.Sprintf("%d", second)
}

func (store *ledisStore) Ttl(key string) string {
	store.lock.Lock()
	defer store.lock.Unlock()

	if _, ok := store.data[key]; !ok {
		return "key not found"
	}
	if _, ok := store.expire[key]; !ok {
		return "-1"
	}

	return fmt.Sprintf("%d", store.expire[key]-time.Now().Unix())
}
