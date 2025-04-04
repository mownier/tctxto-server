# tctxto

A game server for TicTacToe


## How to generate go files from the proto file

```
$ protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative server/tctxto.proto
```