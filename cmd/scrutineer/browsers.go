package main

import (
	"fmt"
	"os"

	"github.com/scrutineer/scrutineer/core/exitcode"
)

func cmdBrowsers(args []string) int {
	if len(args) < 1 {
		printBrowsersUsage()
		return exitcode.ConfigError
	}

	switch args[0] {
	case "install":
		return cmdBrowsersInstall(args[1:])
	case "list":
		return cmdBrowsersList()
	case "help", "--help", "-h":
		printBrowsersUsage()
		return exitcode.OK
	default:
		fmt.Fprintf(os.Stderr, "unknown browsers command: %s\n\n", args[0])
		printBrowsersUsage()
		return exitcode.ConfigError
	}
}

func cmdBrowsersInstall(args []string) int {
	// TODO: Phase 7 — implement browser download from Playwright CDN
	fmt.Println("Browser installation not yet implemented (Phase 7)")
	fmt.Println("This will download Playwright's patched Chromium, Firefox, and WebKit builds.")
	return exitcode.OK
}

func cmdBrowsersList() int {
	// TODO: Phase 7 — list installed browsers
	fmt.Println("Browser listing not yet implemented (Phase 7)")
	return exitcode.OK
}

func printBrowsersUsage() {
	fmt.Println(`Usage: scrutineer browsers <command>

Commands:
  install    Download browser binaries (Chromium, Firefox, WebKit)
  list       List installed browsers
  help       Show this help`)
}
