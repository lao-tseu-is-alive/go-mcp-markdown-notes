// Package notesclient provides a reusable Connect RPC client for the
// notes service.
//
// It supports bearer token authentication and is configurable via options
// such as WithHTTPClient and WithTimeout. Both the MCP server
// (pkg/mcpnotes) and the CLI client (cmd/notes-client) use this package.
package notesclient
