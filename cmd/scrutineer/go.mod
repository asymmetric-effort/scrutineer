module github.com/scrutineer/scrutineer/cmd/scrutineer

go 1.26.2

require (
	github.com/scrutineer/scrutineer/connector/browser v0.0.0
	github.com/scrutineer/scrutineer/connector/cli v0.0.0
	github.com/scrutineer/scrutineer/connector/grpc v0.0.0
	github.com/scrutineer/scrutineer/connector/http v0.0.0
	github.com/scrutineer/scrutineer/connector/ssh v0.0.0
	github.com/scrutineer/scrutineer/core v0.0.0
)

require (
	golang.org/x/crypto v0.36.0 // indirect
	golang.org/x/net v0.35.0 // indirect
	golang.org/x/sys v0.31.0 // indirect
	golang.org/x/text v0.23.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250218202821-56aae31c358a // indirect
	google.golang.org/grpc v1.72.0 // indirect
	google.golang.org/protobuf v1.36.6 // indirect
)

replace (
	github.com/scrutineer/scrutineer/connector/browser => ../../connector/browser
	github.com/scrutineer/scrutineer/connector/cli => ../../connector/cli
	github.com/scrutineer/scrutineer/connector/grpc => ../../connector/grpc
	github.com/scrutineer/scrutineer/connector/http => ../../connector/http
	github.com/scrutineer/scrutineer/connector/ssh => ../../connector/ssh
	github.com/scrutineer/scrutineer/core => ../../core
	github.com/scrutineer/scrutineer/fuzz => ../../fuzz
	github.com/scrutineer/scrutineer/loadtest => ../../loadtest
)
