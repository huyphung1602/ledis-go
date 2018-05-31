package handlers

import (
	"encoding/gob"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	shellquote "github.com/kballard/go-shellquote"
)

type LedisHandler struct {
}

var store *LedisStore

type ledisType int

const (
	TypeSet ledisType = iota
	TypeList
	TypeString
)

type LedisData struct {
	DataType   ledisType
	SetData    *map[string]bool
	ListData   *[]string
	StringData *string
}

type LedisStore struct {
	Data       map[string]LedisData
	ExpireTime map[string]int64
	lock       *sync.RWMutex
}

func InitStore() {
	store = &LedisStore{
		Data:       make(map[string]LedisData),
		ExpireTime: make(map[string]int64),
		lock:       &sync.RWMutex{},
	}
}

func ExpiredCleaner() {
	for {
		time.Sleep(500 * time.Millisecond)
		store.lock.RLock()

		timeNow := time.Now().Unix()
		for key, val := range store.ExpireTime {
			if val-timeNow <= 0 {
				delete(store.ExpireTime, key)
				delete(store.Data, key)
			}
		}
		store.lock.RUnlock()
	}
}

func (h *LedisHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	body, _ := ioutil.ReadAll(r.Body)
	bodyStr := string(body[:])

	setHTTPStatus(w)
	cmd, err := parseCommand(bodyStr)
	if err != nil {
		respError(w, err)
		return
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
			respError(w, fmt.Errorf("LPOP expects 1 argument"))
			return
		}
		writeBody(w, store.Lpop(cmd.Args[0]))
	case "RPOP":
		if len(cmd.Args) != 1 {
			respError(w, fmt.Errorf("RPOP expects 1 argument"))
			return
		}
		writeBody(w, store.Rpop(cmd.Args[0]))
	case "LRANGE":
		if len(cmd.Args) != 3 {
			respError(w, fmt.Errorf("LRANGE expects 3 arguments"))
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
			respError(w, fmt.Errorf("Error when parsing seconds"))
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
	case "SAVE":
		writeBody(w, store.Save())
	case "RESTORE":
		writeBody(w, store.Restore())
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
	if len(args) < 1 {
		return nil, fmt.Errorf("empty command")
	}

	return &command{Name: args[0], Args: args[1:]}, nil
}

func setHTTPStatus(w http.ResponseWriter) {
	w.Header().Add("Access-Control-Allow-Origin", `*`)
	w.Header().Add("Access-Control-Allow-Methods", `GET, POST, PUT, DELETE, OPTIONS`)
}

func (store *LedisStore) Get(key string) string {
	store.lock.RLock()
	defer store.lock.RUnlock()

	storeVal, ok := store.Data[key]
	if !ok {
		return "key not found"
	}
	if storeVal.DataType != TypeString {
		return "WRONGTYPE Operation against a key holding the wrong kind of value"
	}

	return *storeVal.StringData
}

func (store *LedisStore) Set(key string, val string) {
	store.lock.Lock()
	defer store.lock.Unlock()

	// set always success, it even overwrite other data types
	store.Data[key] = LedisData{
		DataType:   TypeString,
		SetData:    nil,
		ListData:   nil,
		StringData: &val}
	// remove the expire if exist
	delete(store.ExpireTime, key)
}

func (store *LedisStore) Llen(key string) string {
	store.lock.Lock()
	defer store.lock.Unlock()

	if store.Data[key].DataType != TypeList {
		return "WRONGTYPE Operation against a key holding the wrong kind of value"
	}

	return fmt.Sprintf("%d", len(*store.Data[key].ListData))
}

func (store *LedisStore) Rpush(key string, values []string) string {
	store.lock.Lock()
	defer store.lock.Unlock()

	storeVal, ok := store.Data[key]
	if ok {
		if storeVal.DataType != TypeList {
			return "WRONGTYPE Operation against a key holding the wrong kind of value"
		}

		// append value
		for _, val := range values {
			*store.Data[key].ListData = append(*store.Data[key].ListData, val)
		}
		return fmt.Sprintf("%d", len(*storeVal.ListData))
	}

	// create the list
	store.Data[key] = LedisData{
		DataType:   TypeList,
		SetData:    nil,
		ListData:   &values,
		StringData: nil}
	return fmt.Sprintf("%d", len(values))
}

func (store *LedisStore) Lpop(key string) string {
	store.lock.Lock()
	defer store.lock.Unlock()

	// check if key is exist
	storeVal, ok := store.Data[key]
	if !ok {
		return "key not found"
	}

	if storeVal.DataType != TypeList {
		return "WRONGTYPE Operation against a key holding the wrong kind of value"
	}

	// else, lpop
	if len(*storeVal.ListData) == 0 {
		return "(nil)"
	}
	retVal := (*storeVal.ListData)[0]
	*storeVal.ListData = append((*storeVal.ListData)[:0], (*storeVal.ListData)[1:]...)
	return retVal
}

func (store *LedisStore) Rpop(key string) string {
	store.lock.Lock()
	defer store.lock.Unlock()

	// check if key is exist
	storeVal, ok := store.Data[key]
	if !ok {
		return "key not found"
	}

	// if key is not list, return wrong type
	if storeVal.DataType != TypeList {
		return "WRONGTYPE Operation against a key holding the wrong kind of value"
	}

	// else, rpop
	if len(*storeVal.ListData) == 0 {
		return "(nil)"
	}
	lastIdx := len(*storeVal.ListData) - 1
	retVal := (*storeVal.ListData)[lastIdx]
	*storeVal.ListData = append((*storeVal.ListData)[:lastIdx], (*storeVal.ListData)[lastIdx+1:]...)
	return retVal
}

func (store *LedisStore) Lrange(key string, start, stop uint64) string {
	store.lock.Lock()
	defer store.lock.Unlock()

	// check if key is exist
	storeVal, ok := store.Data[key]
	if !ok {
		return "key not found"
	}

	// if key is not list, return wrong type
	if storeVal.DataType != TypeList {
		return "WRONGTYPE Operation against a key holding the wrong kind of value"
	}

	lenListData := uint64(len(*storeVal.ListData))
	if lenListData == 0 {
		return "(nil)"
	}

	stopIdx := stop
	if stopIdx >= lenListData {
		stopIdx = lenListData
	}

	retStr := ""
	for i := start; i < stopIdx; i++ {
		retStr += (*storeVal.ListData)[i] + "\r\n"
	}
	return retStr
}

func (store *LedisStore) Sadd(key string, values []string) string {
	store.lock.Lock()
	defer store.lock.Unlock()

	count := 0
	storeVal, ok := store.Data[key]
	if ok {
		if storeVal.DataType != TypeSet {
			return "WRONGTYPE Operation against a key holding the wrong kind of value"
		}

		// add item to set
		setVals := *storeVal.SetData
		for _, val := range values {
			if _, ok = setVals[val]; !ok {
				count++
				setVals[val] = true
			}
		}

		storeVal.SetData = &setVals
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
	store.Data[key] = LedisData{
		DataType:   TypeSet,
		SetData:    &setVals,
		ListData:   nil,
		StringData: nil}
	return fmt.Sprintf("%d", len(values))
}

func (store *LedisStore) Scard(key string) string {
	store.lock.Lock()
	defer store.lock.Unlock()

	count := 0
	storeVal, ok := store.Data[key]
	if !ok {
		return "key not found"
	}
	if storeVal.DataType != TypeSet {
		return "WRONGTYPE Operation against a key holding the wrong kind of value"
	}

	for _ = range *storeVal.SetData {
		count++
	}
	return fmt.Sprintf("%d", count)
}

func (store *LedisStore) Smembers(key string) string {
	store.lock.Lock()
	defer store.lock.Unlock()

	resStr := ""
	storeVal, ok := store.Data[key]
	if !ok {
		return "key not found"
	}
	if storeVal.DataType != TypeSet {
		return "WRONGTYPE Operation against a key holding the wrong kind of value"
	}

	if len(*storeVal.SetData) == 0 {
		return "(empty set)"
	}

	for key := range *storeVal.SetData {
		resStr += key + "\r\n"
	}
	return resStr
}

func (store *LedisStore) Srem(key string, values []string) string {
	store.lock.Lock()
	defer store.lock.Unlock()

	count := 0
	storeVal, ok := store.Data[key]
	if !ok {
		return "key not found"
	}
	if storeVal.DataType != TypeSet {
		return "WRONGTYPE Operation against a key holding the wrong kind of value"
	}

	if len(*storeVal.SetData) == 0 {
		return "0"
	}

	for _, val := range values {
		if _, ok := (*storeVal.SetData)[val]; ok {
			count++
			delete(*storeVal.SetData, val)
		}
	}

	return fmt.Sprintf("%d", count)
}

func (store *LedisStore) Sinter(keys []string) string {
	store.lock.Lock()
	defer store.lock.Unlock()

	firstSet := *store.Data[keys[0]].SetData
	for key := range firstSet {
		firstSet[key] = true
	}
	for _, key := range keys {
		storeVal, ok := store.Data[key]
		if !ok {
			return fmt.Sprintf("key not found: %s", key)
		}
		if storeVal.DataType != TypeSet {
			return fmt.Sprintf("WRONGTYPE Operation against a key: %s holding the wrong kind of value", key)
		}

		for setVal := range firstSet {
			if _, ok := (*storeVal.SetData)[setVal]; !ok {
				firstSet[setVal] = false
			}
		}
	}

	resStr := ""
	for key := range firstSet {
		if firstSet[key] == true {
			resStr += key + "\r\n"
		}
	}

	if resStr == "" {
		return "empty"
	}
	return resStr
}

func (store *LedisStore) Keys() string {
	store.lock.Lock()
	defer store.lock.Unlock()

	valRet := ""
	for key := range store.Data {
		valRet += key + "\r\n"
	}

	if valRet == "" {
		return "empty"
	}
	return valRet
}

func (store *LedisStore) Del(key string) string {
	store.lock.Lock()
	defer store.lock.Unlock()

	if _, ok := store.Data[key]; !ok {
		return "key not found"
	}

	delete(store.Data, key)
	return "1"
}

func (store *LedisStore) Flushdb() string {
	store.lock.Lock()
	defer store.lock.Unlock()
	for key := range store.Data {
		delete(store.Data, key)
	}
	return "OK"
}

func (store *LedisStore) Expire(key string, second int64) string {
	store.lock.Lock()
	defer store.lock.Unlock()

	if _, ok := store.Data[key]; !ok {
		return "key not found"
	}

	store.ExpireTime[key] = time.Now().Unix() + second
	return fmt.Sprintf("%d", second)
}

func (store *LedisStore) Ttl(key string) string {
	store.lock.Lock()
	defer store.lock.Unlock()

	if _, ok := store.Data[key]; !ok {
		return "key not found"
	}
	if _, ok := store.ExpireTime[key]; !ok {
		return "-1"
	}

	return fmt.Sprintf("%d", store.ExpireTime[key]-time.Now().Unix())
}

func (store *LedisStore) Save() string {
	store.lock.Lock()
	defer store.lock.Unlock()
	encodeFile, err := os.Create("accounts.gob")
	if err != nil {
		return err.Error()
	}

	e := gob.NewEncoder(encodeFile)

	err = e.Encode(store)
	if err != nil {
		return err.Error()
	}

	return "OK"
}

func (store *LedisStore) Restore() string {
	store.lock.Lock()
	defer store.lock.Unlock()

	// Open a RO file
	decodeFile, err := os.Open("accounts.gob")
	if err != nil {
		return err.Error()
	}
	defer decodeFile.Close()

	var decodedMap LedisStore
	d := gob.NewDecoder(decodeFile)

	// Decoding the serialized data
	err = d.Decode(&decodedMap)
	if err != nil {
		return err.Error()
	}

	// restore all keys in the decodedMap
	for key, val := range decodedMap.Data {
		delete(store.Data, key)
		store.Data[key] = val
	}
	for key, val := range decodedMap.ExpireTime {
		delete(store.ExpireTime, key)
		store.ExpireTime[key] = val
	}

	return "OK"
}
