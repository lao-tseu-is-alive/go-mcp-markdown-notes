package main

import (
	"fmt"
	"github.com/lao-tseu-is-alive/go-mcp-markdown-notes/pkg/version"
)

func main() {
	fmt.Printf("🚀 Starting %s v%s (rev: %s, build: %s)\n",
		version.AppName, version.Version, version.Revision, version.BuildStamp)
	// TODO: implement notes-server
	fmt.Println("⚠️  notes-server is not yet implemented")
}
