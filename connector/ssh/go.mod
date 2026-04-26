module github.com/scrutineer/scrutineer/connector/ssh

go 1.26.2

require (
	github.com/scrutineer/scrutineer/core v0.0.0
	golang.org/x/crypto v0.36.0
)

require golang.org/x/sys v0.31.0 // indirect

replace github.com/scrutineer/scrutineer/core => ../../core
