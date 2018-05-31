# Introduction
- **Ledis**: a simple, stripped down version of a Redis server, with these functionalities:
    + Data structures: String, List, Set
    + Special features: Expire, snapshots
    + A simple web frontend CLI (similar to redis-cli), located at: `cli/ledis-cli.html`

- To Run:
```
$ go get -u github.com/golang/dep/cmd/dep
$ dep ensure
$ gin -a 8080 run main.go
```

- Test Coverage:
```
$ ./test.sh
?       github.com/zealotnt/ledis-go    [no test files]
ok      github.com/zealotnt/ledis-go/handlers   4.059s  coverage: 98.5% of statements
github.com/zealotnt/ledis-go/handlers/ledis_handler.go:44:  InitStore   100.0%
github.com/zealotnt/ledis-go/handlers/ledis_handler.go:52:  ExpiredCleaner  100.0%
github.com/zealotnt/ledis-go/handlers/ledis_handler.go:68:  ServeHTTP   100.0%
github.com/zealotnt/ledis-go/handlers/ledis_handler.go:211: writeBody   100.0%
github.com/zealotnt/ledis-go/handlers/ledis_handler.go:215: respError   100.0%
github.com/zealotnt/ledis-go/handlers/ledis_handler.go:219: parseCommand    83.3%
github.com/zealotnt/ledis-go/handlers/ledis_handler.go:231: setHTTPStatus   100.0%
github.com/zealotnt/ledis-go/handlers/ledis_handler.go:236: Get     100.0%
github.com/zealotnt/ledis-go/handlers/ledis_handler.go:251: Set     100.0%
github.com/zealotnt/ledis-go/handlers/ledis_handler.go:265: Llen        100.0%
github.com/zealotnt/ledis-go/handlers/ledis_handler.go:276: Rpush       100.0%
github.com/zealotnt/ledis-go/handlers/ledis_handler.go:302: Lpop        100.0%
github.com/zealotnt/ledis-go/handlers/ledis_handler.go:325: Rpop        100.0%
github.com/zealotnt/ledis-go/handlers/ledis_handler.go:350: Lrange      100.0%
github.com/zealotnt/ledis-go/handlers/ledis_handler.go:382: Sadd        100.0%
github.com/zealotnt/ledis-go/handlers/ledis_handler.go:422: Scard       100.0%
github.com/zealotnt/ledis-go/handlers/ledis_handler.go:441: Smembers    100.0%
github.com/zealotnt/ledis-go/handlers/ledis_handler.go:464: Srem        100.0%
github.com/zealotnt/ledis-go/handlers/ledis_handler.go:491: Sinter      100.0%
github.com/zealotnt/ledis-go/handlers/ledis_handler.go:528: Keys        100.0%
github.com/zealotnt/ledis-go/handlers/ledis_handler.go:543: Del     100.0%
github.com/zealotnt/ledis-go/handlers/ledis_handler.go:555: Flushdb     100.0%
github.com/zealotnt/ledis-go/handlers/ledis_handler.go:564: Expire      100.0%
github.com/zealotnt/ledis-go/handlers/ledis_handler.go:576: Ttl     100.0%
github.com/zealotnt/ledis-go/handlers/ledis_handler.go:590: Save        80.0%
github.com/zealotnt/ledis-go/handlers/ledis_handler.go:608: Restore     88.9%
total:                              (statements)    98.5%
```
