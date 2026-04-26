package main

import (
	"fmt"
	"runtime"
)

func printVersion() {
	fmt.Printf("scrutineer %s\n", version)
	fmt.Printf("  go:       %s\n", runtime.Version())
	fmt.Printf("  os/arch:  %s/%s\n", runtime.GOOS, runtime.GOARCH)
}
