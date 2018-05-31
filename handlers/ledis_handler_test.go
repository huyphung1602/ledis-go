package handlers_test

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/parnurzeal/gorequest"
	"github.com/zealotnt/ledis-go/handlers"

	. "github.com/onsi/gomega"
)

var serverUrl string

func SendCommand(cmd string) string {
	_, body, errs := gorequest.New().Post(serverUrl).Type("text").SendString(cmd).End()
	if errs != nil {
		panic(errs)
	}
	return body
}

type ValidateExactTest struct {
	command  string
	expect   string
	testName string
}

type ValidateContainTest struct {
	command  string
	expects  []string
	testName string
}

func TestExpire(t *testing.T) {
	handlers.InitStore()
	handler := &handlers.LedisHandler{}
	server := httptest.NewServer(handler)
	go handlers.ExpiredCleaner()
	defer server.Close()
	serverUrl = server.URL
	g := NewGomegaWithT(t)

	body := SendCommand(`SET testkey 123`)
	g.Expect(body).To(Equal("OK"))
	body = SendCommand(`EXPIRE testkey 100`)
	g.Expect(body).To(Equal("100"))
	time.Sleep(1 * time.Second)
	body = SendCommand(`TTL testkey`)
	g.Expect(body).To(Equal("99"), "Test TTL timing substract")

	// after expire time, key should be removed
	body = SendCommand(`EXPIRE testkey 2`)
	g.Expect(body).To(Equal("2"))
	time.Sleep(3 * time.Second)
	body = SendCommand(`GET testkey`)
	g.Expect(body).To(Equal("key not found"), "Test TTL expired")
}

func TestLedisOps(t *testing.T) {
	handlers.InitStore()
	handler := &handlers.LedisHandler{}
	server := httptest.NewServer(handler)
	defer server.Close()
	serverUrl = server.URL
	g := NewGomegaWithT(t)

	tests := []ValidateExactTest{
		{`SET testkey 123`, "OK", ""},
		{`GET testkey`, "123", ""},
		{`GET testkey1`, "key not found", ""},
		{`RPUSH testlist 1 2 3 4`, "4", ""},
		{`RPUSH testlist 5 6`, "6", ""},
		{`RPUSH testkey 1 2 3 4`, "WRONGTYPE Operation against a key holding the wrong kind of value", ""},
		{`GET testlist`, "WRONGTYPE Operation against a key holding the wrong kind of value", ""},
		{`LLEN testlist`, "6", ""},
		{`LLEN testkey`, "WRONGTYPE Operation against a key holding the wrong kind of value", ""},
		{`LPOP testlist`, "1", ""},
		{`LPOP no-exist`, "key not found", ""},
		{`LPOP testkey`, "WRONGTYPE Operation against a key holding the wrong kind of value", ""},
		{`RPOP testlist`, "6", ""},
		{`RPOP no-exist`, "key not found", ""},
		{`RPOP testkey`, "WRONGTYPE Operation against a key holding the wrong kind of value", ""},
		{`LLEN testlist`, "4", "Test LLEN"},
		{`LRANGE testlist 0 1000`, "2\r\n3\r\n4\r\n5\r\n", "Test LRANGE"},
		{`LRANGE no-exist 0 1000`, "key not found", ""},
		{`LRANGE testkey 1 2`, "WRONGTYPE Operation against a key holding the wrong kind of value", ""},
		{`RPOP testlist`, "5", ""},
		{`RPOP testlist`, "4", ""},
		{`LPOP testlist`, "2", ""},
		{`LPOP testlist`, "3", ""},
		{`LPOP testlist`, "(nil)", ""},
		{`RPOP testlist`, "(nil)", ""},
		{`LRANGE testlist 1 2`, "(nil)", ""},
		{`SADD testset 1 2 3`, "3", "Test SADD"},
		{`SADD testkey 1 2 3`, "WRONGTYPE Operation against a key holding the wrong kind of value", ""},
		{`SCARD testset`, "3", "Test SCARD"},

		// SREM
		{`SREM testset 1`, "1", "Test SREM"},
		{`SREM no-exist 1`, "key not found", ""},
		{`SMEMBERS no-exist`, "key not found", ""},
		{`SMEMBERS testkey`, "WRONGTYPE Operation against a key holding the wrong kind of value", ""},
		{`SREM testkey 1`, "WRONGTYPE Operation against a key holding the wrong kind of value", ""},
		{`SCARD testset`, "2", "Test SCARD after remove elem from set"},
		{`SREM testset 2 3`, "2", "Remove all other elem in set"},
		{`SMEMBERS testset`, "(empty set)", ""},
		{`SREM testset a b c`, "0", "If no elem in set, return 0"},
		{`SADD testset x y z`, "3", "Append item to testset"},
		{`SCARD testset`, "3", "Test SCARD after append item to set"},
		{`SCARD no-exist`, "key not found", ""},
		{`SCARD testkey`, "WRONGTYPE Operation against a key holding the wrong kind of value", ""},

		// SINTER
		{`SADD testset1 a 1 2 3`, "4", "Prep Test SINTER 1"},
		{`SADD testset2 a 4 5 6`, "4", "Prep Test SINTER 2"},
		{`SADD testset3 a 7 8 9`, "4", "Prep Test SINTER 3"},
		{`SINTER testset1 testset2 testset3`, "a\r\n", "Test SINTER"},
		{`SINTER testset1 testset2 testset3 testkey`, "WRONGTYPE Operation against a key: testkey holding the wrong kind of value", ""},
		{`SINTER testset1 testset2 testset3 no-exist`, "key not found: no-exist", ""},
		{`SINTER testset1 testset2 testset3 testset`, "empty", ""},

		{`DEL testkey`, "1", "Test DEL"},
		{`DEL no-exist`, "key not found", ""},

		{`EXPIRE no-exist 100`, "key not found", ""},
		{`TTL no-exist`, "key not found", ""},
		{`TTL testset1`, "-1", ""},

		{`EXPIRE testset 100`, "100", ""},
		{`SAVE`, "OK", "Test SAVE"},
		{`FLUSHDB`, "OK", "Test FLUSHDB"},
		{`KEYS`, "empty", ""},
		{`RESTORE`, "OK", "Test RESTORE"},
		{`TTL testset`, "100", ""},
	}
	for _, test := range tests {
		body := SendCommand(test.command)
		g.Expect(body).To(Equal(test.expect), test.testName)
	}

	// unorderred item
	testContains := []ValidateContainTest{
		{`SMEMBERS testset`, []string{"x", "y", "z"}, "Test SMEMBERS"},
		{`KEYS`, []string{"testset", "testset1", "testset2", "testset3", "testlist"}, "Test KEYS"},
	}

	for _, test := range testContains {
		body := SendCommand(test.command)
		for _, expect := range test.expects {
			g.Expect(body).To(ContainSubstring(expect), test.testName)
		}
	}
}

func TestInvalidCommand(t *testing.T) {
	handlers.InitStore()
	handler := &handlers.LedisHandler{}
	server := httptest.NewServer(handler)
	defer server.Close()
	serverUrl = server.URL
	g := NewGomegaWithT(t)

	type InvalidCommand struct {
		command string
		errMsg  string
	}
	tests := []InvalidCommand{
		{"", "empty command"},
		{"GET", "GET expects 1 argument"},
		{"SET somekey", "SET expects 2 arguments"},
		{"LLEN", "LLEN expects 1 argument"},
		{"RPUSH somekey", "RPUSH expects at least 2 arguments"},
		{"LPOP", "LPOP expects 1 argument"},
		{"RPOP", "RPOP expects 1 argument"},
		{"LRANGE", "LRANGE expects 3 arguments"},
		{"LRANGE somekey -1 1", "Error when parsing start"},
		{"LRANGE somekey 1 -1", "Error when parsing end"},
		{"SADD somekey", "SADD expects at least 2 arguments"},
		{"SCARD", "SCARD expects 1 arguments"},
		{"SMEMBERS", "SMEMBERS expects 1 arguments"},
		{"SREM somekey", "SREM expects at least 2 arguments"},
		{"SINTER somekey", "SINTER expects at least 2 arguments"},
		{"DEL", "DEL expects 1 argument"},
		{"EXPIRE", "EXPIRE expects 2 arguments"},
		{"EXPIRE somekey abc", "Error when parsing seconds"},
		{"EXPIRE somekey -1", "Second should be a positive number"},
		{"TTL", "TTL expects 1 argument"},
		{"some-invalid-command", "unkonwn command: some-invalid-command"},
	}

	for _, test := range tests {
		body := SendCommand(test.command)
		g.Expect(body).To(ContainSubstring(test.errMsg))
	}
}
