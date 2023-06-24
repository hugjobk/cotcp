# Golang TCP Client-Server library allowing to send requests and responses concurrently per connection.

## Test cases

### To start CoTCP server:
```
go test -v -run ^TestServer_ListenAndServe$ github.com/hugjobk/cotcp
```

### To check if server is running:
```
go test -v -run ^TestClient_Ping$ github.com/hugjobk/cotcp
```

### To send message with reply from server:
```
go test -v -run ^TestClient_Send$ github.com/hugjobk/cotcp
```

### To send message without reply from server:
```
go test -v -run ^TestClient_SendNoReply$ github.com/hugjobk/cotcp
```

## Benchmark

### See benchamrk results in the paper `CoTCP - A New Approach to The Concurrent TCP`

### To run the benchmark:
```
go run benchmark/main.go
```
