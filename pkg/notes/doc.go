// Package notes contains the domain model, business logic (Service), transport adapter (ConnectServer),
// and PostgreSQL repository for the go-mcp-markdown-notes note management service.
//
// The public contract lives in proto/notes/v1/notes.proto (generates the connect/pb types).
// The internal Note model + raw SQL are maintained by hand. Named struct scanning (pgx)
// based on `db` tags reduces the cost of adding columns. See AGENTS.md section
// "Evolving the Proto / Note Contract" and the comments in model.go + sql.go.
package notes
