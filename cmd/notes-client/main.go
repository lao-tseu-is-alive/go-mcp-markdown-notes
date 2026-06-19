package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"connectrpc.com/connect"
	v1 "github.com/lao-tseu-is-alive/go-mcp-markdown-notes/gen/notes/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func main() {
	// Root/global flags
	serverURL := flag.String("server", "http://127.0.0.1:8080", "Server base URL (or -s)")
	flag.StringVar(serverURL, "s", "http://127.0.0.1:8080", "Server base URL")

	token := flag.String("token", "", "Bearer token for authorization (or -t)")
	flag.StringVar(token, "t", "", "Bearer token for authorization")

	format := flag.String("format", "text", "Output format: text or json (or -f)")
	flag.StringVar(format, "f", "text", "Output format: text or json")

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s:\n", os.Args[0])
		fmt.Fprintf(flag.CommandLine.Output(), "  notes-client [global flags] <subcommand> [subcommand flags]\n\n")
		fmt.Fprintf(flag.CommandLine.Output(), "Global flags:\n")
		flag.PrintDefaults()
		fmt.Fprintf(flag.CommandLine.Output(), "\nSubcommands:\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  create   Create a new note (flags: -title, -body, -category, -tags, -status)\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  get      Retrieve a note by ID\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  list     List recent notes\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  search   Search notes\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  tag      Add tags to a note\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  update   Update a note (flags: -id, -title, -body, -category, -tags, -status)\n")
	}

	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		flag.Usage()
		os.Exit(1)
	}

	subcommand := args[0]
	subArgs := args[1:]

	// Fallback token authentication lookup order:
	// 1. Explicit CLI Flag
	// 2. NOTES_TOKEN env variable
	// 3. NOTES_DEV_TOKEN env variable
	authToken := *token
	if authToken == "" {
		authToken = os.Getenv("NOTES_TOKEN")
	}
	if authToken == "" {
		authToken = os.Getenv("NOTES_DEV_TOKEN")
	}

	client := newNotesClient(*serverURL, authToken)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	switch subcommand {
	case "create":
		fs := flag.NewFlagSet("create", flag.ExitOnError)
		title := fs.String("title", "", "Title of the note (required)")
		body := fs.String("body", "", "Markdown content of the note (required)")
		category := fs.String("category", "", "Category of the note")
		tagsStr := fs.String("tags", "", "Comma-separated list of tags")
		statusStr := fs.String("status", "", "Lifecycle status: draft, active, final, or archived")
		fs.Parse(subArgs)

		if *title == "" || *body == "" {
			fmt.Fprintln(os.Stderr, "Error: -title and -body are required.")
			fs.Usage()
			os.Exit(1)
		}

		var tags []string
		if *tagsStr != "" {
			for _, t := range strings.Split(*tagsStr, ",") {
				if t = strings.TrimSpace(t); t != "" {
					tags = append(tags, t)
				}
			}
		}

		req := &v1.CreateNoteRequest{
			Title:        *title,
			BodyMarkdown: *body,
			Category:     *category,
			Tags:         tags,
		}
		if *statusStr != "" {
			st, err := parseNoteStatus(*statusStr)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			req.Status = &st
		}

		resp, err := client.CreateNote(ctx, connect.NewRequest(req))
		if err != nil {
			printError(err, *format)
			os.Exit(1)
		}
		printResponse(resp.Msg, *format)

	case "get":
		fs := flag.NewFlagSet("get", flag.ExitOnError)
		id := fs.String("id", "", "ID of the note")
		fs.Parse(subArgs)

		noteID := *id
		if noteID == "" && fs.NArg() > 0 {
			noteID = fs.Arg(0)
		}

		if noteID == "" {
			fmt.Fprintln(os.Stderr, "Error: -id or positional note ID is required.")
			fs.Usage()
			os.Exit(1)
		}

		resp, err := client.GetNote(ctx, connect.NewRequest(&v1.GetNoteRequest{
			Id: noteID,
		}))
		if err != nil {
			printError(err, *format)
			os.Exit(1)
		}
		printResponse(resp.Msg, *format)

	case "list":
		fs := flag.NewFlagSet("list", flag.ExitOnError)
		limit := fs.Int("limit", 10, "Maximum number of notes to retrieve")
		fs.Parse(subArgs)

		resp, err := client.ListRecentNotes(ctx, connect.NewRequest(&v1.ListRecentNotesRequest{
			Limit: int32(*limit),
		}))
		if err != nil {
			printError(err, *format)
			os.Exit(1)
		}
		printResponse(resp.Msg, *format)

	case "search":
		fs := flag.NewFlagSet("search", flag.ExitOnError)
		query := fs.String("query", "", "Search query")
		category := fs.String("category", "", "Search category")
		tagsStr := fs.String("tags", "", "Comma-separated list of tags")
		limit := fs.Int("limit", 10, "Maximum number of results")
		fs.Parse(subArgs)

		var tags []string
		if *tagsStr != "" {
			for _, t := range strings.Split(*tagsStr, ",") {
				if t = strings.TrimSpace(t); t != "" {
					tags = append(tags, t)
				}
			}
		}

		l := int32(*limit)
		resp, err := client.SearchNotes(ctx, connect.NewRequest(&v1.SearchNotesRequest{
			Query:    *query,
			Category: *category,
			Tags:     tags,
			Limit:    &l,
		}))
		if err != nil {
			printError(err, *format)
			os.Exit(1)
		}
		printResponse(resp.Msg, *format)

	case "tag":
		fs := flag.NewFlagSet("tag", flag.ExitOnError)
		id := fs.String("id", "", "ID of the note (required)")
		tagsStr := fs.String("tags", "", "Comma-separated list of tags to add (required)")
		fs.Parse(subArgs)

		noteID := *id
		if noteID == "" && fs.NArg() > 0 {
			noteID = fs.Arg(0)
		}

		if noteID == "" || *tagsStr == "" {
			fmt.Fprintln(os.Stderr, "Error: -id and -tags are required.")
			fs.Usage()
			os.Exit(1)
		}

		var tags []string
		for _, t := range strings.Split(*tagsStr, ",") {
			if t = strings.TrimSpace(t); t != "" {
				tags = append(tags, t)
			}
		}

		resp, err := client.AddTags(ctx, connect.NewRequest(&v1.AddTagsRequest{
			NoteId: noteID,
			Tags:   tags,
		}))
		if err != nil {
			printError(err, *format)
			os.Exit(1)
		}
		printResponse(resp.Msg, *format)

	case "update":
		fs := flag.NewFlagSet("update", flag.ExitOnError)
		id := fs.String("id", "", "ID of the note (required)")
		title := fs.String("title", "", "Updated title")
		body := fs.String("body", "", "Updated body content")
		category := fs.String("category", "", "Updated category")
		tagsStr := fs.String("tags", "", "Comma-separated list of tags")
		statusStr := fs.String("status", "", "Lifecycle status: draft, active, final, or archived")
		fs.Parse(subArgs)

		noteID := *id
		if noteID == "" && fs.NArg() > 0 {
			noteID = fs.Arg(0)
		}

		if noteID == "" {
			fmt.Fprintln(os.Stderr, "Error: -id is required.")
			fs.Usage()
			os.Exit(1)
		}

		var tags []string
		var hasTags bool
		if *tagsStr != "" {
			hasTags = true
			for _, t := range strings.Split(*tagsStr, ",") {
				if t = strings.TrimSpace(t); t != "" {
					tags = append(tags, t)
				}
			}
		}

		req := &v1.UpdateNoteRequest{
			NoteId:       noteID,
			Title:        *title,
			BodyMarkdown: *body,
			Category:     *category,
		}
		if hasTags {
			req.Tags = tags
		}
		if *statusStr != "" {
			st, err := parseNoteStatus(*statusStr)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			req.Status = &st
		}

		resp, err := client.UpdateNote(ctx, connect.NewRequest(req))
		if err != nil {
			printError(err, *format)
			os.Exit(1)
		}
		printResponse(resp.Msg, *format)

	default:
		fmt.Fprintf(os.Stderr, "Unknown subcommand: %s\n", subcommand)
		flag.Usage()
		os.Exit(1)
	}
}

func printNoteText(note *v1.Note) {
	if note == nil {
		return
	}
	fmt.Printf("\033[1;36m+----------------------------------------------------------------------+\033[0m\n")
	fmt.Printf("\033[1;36m| NOTE ID:\033[0m %-60s \033[1;36m|\033[0m\n", note.Id)
	fmt.Printf("\033[1;36m| TITLE:\033[0m   \033[1;37m%-60s\033[0m \033[1;36m|\033[0m\n", note.Title)
	if note.Category != "" {
		fmt.Printf("\033[1;36m| CAT:\033[0m     \033[1;34m%-60s\033[0m \033[1;36m|\033[0m\n", note.Category)
	}
	if len(note.Tags) > 0 {
		fmt.Printf("\033[1;36m| TAGS:\033[0m    \033[1;35m%-60s\033[0m \033[1;36m|\033[0m\n", strings.Join(note.Tags, ", "))
	}
	if st := noteStatusToString(note.Status); st != "" {
		fmt.Printf("\033[1;36m| STATUS:\033[0m  \033[1;33m%-60s\033[0m \033[1;36m|\033[0m\n", st)
	}
	fmt.Printf("\033[1;36m| CREATED:\033[0m %-60s \033[1;36m|\033[0m\n", protoTimestampToString(note.CreatedAt))
	fmt.Printf("\033[1;36m| UPDATED:\033[0m %-60s \033[1;36m|\033[0m\n", protoTimestampToString(note.UpdatedAt))
	if note.OwnerUserId != "" {
		fmt.Printf("\033[1;36m| OWNER ID:\033[0m%-60s \033[1;36m|\033[0m\n", note.OwnerUserId)
	}
	fmt.Printf("\033[1;36m+----------------------------------------------------------------------+\033[0m\n")
	fmt.Printf("\033[1mBody:\033[0m\n%s\n\n", note.BodyMarkdown)
}

func protoTimestampToString(ts *timestamppb.Timestamp) string {
	if ts == nil || ts.CheckValid() != nil {
		return ""
	}
	return ts.AsTime().UTC().Format(time.RFC3339Nano)
}

func parseNoteStatus(s string) (v1.NoteStatus, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "draft":
		return v1.NoteStatus_NOTE_STATUS_DRAFT, nil
	case "active":
		return v1.NoteStatus_NOTE_STATUS_ACTIVE, nil
	case "final":
		return v1.NoteStatus_NOTE_STATUS_FINAL, nil
	case "archived":
		return v1.NoteStatus_NOTE_STATUS_ARCHIVED, nil
	default:
		return v1.NoteStatus_NOTE_STATUS_UNSPECIFIED, fmt.Errorf("unknown status %q: must be draft, active, final, or archived", s)
	}
}

func noteStatusToString(s v1.NoteStatus) string {
	switch s {
	case v1.NoteStatus_NOTE_STATUS_DRAFT:
		return "draft"
	case v1.NoteStatus_NOTE_STATUS_ACTIVE:
		return "active"
	case v1.NoteStatus_NOTE_STATUS_FINAL:
		return "final"
	case v1.NoteStatus_NOTE_STATUS_ARCHIVED:
		return "archived"
	default:
		return ""
	}
}

func printNotesListText(notes []*v1.Note) {
	if len(notes) == 0 {
		fmt.Println("No notes found.")
		return
	}
	fmt.Printf("\033[1;36m%-36s  %-30s  %-10s  %-15s  %-20s\033[0m\n", "NOTE ID", "TITLE", "STATUS", "CATEGORY", "TAGS")
	fmt.Println(strings.Repeat("-", 120))
	for _, n := range notes {
		tags := strings.Join(n.Tags, ",")
		if len(tags) > 20 {
			tags = tags[:17] + "..."
		}
		title := n.Title
		if len(title) > 30 {
			title = title[:27] + "..."
		}
		category := n.Category
		if len(category) > 15 {
			category = category[:12] + "..."
		}
		fmt.Printf("%-36s  %-30s  %-10s  %-15s  %-20s\n", n.Id, title, noteStatusToString(n.Status), category, tags)
	}
	fmt.Println()
}

func printResponse(v any, format string) {
	if format == "json" {
		data, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting JSON: %v\n", err)
			return
		}
		fmt.Println(string(data))
		return
	}

	switch msg := v.(type) {
	case *v1.CreateNoteResponse:
		fmt.Println("\033[1;32mNote created successfully!\033[0m")
		printNoteText(msg.Note)
	case *v1.GetNoteResponse:
		printNoteText(msg.Note)
	case *v1.ListRecentNotesResponse:
		printNotesListText(msg.Notes)
	case *v1.SearchNotesResponse:
		printNotesListText(msg.Notes)
	case *v1.AddTagsResponse:
		fmt.Println("\033[1;32mTags added successfully!\033[0m")
		printNoteText(msg.Note)
	case *v1.UpdateNoteResponse:
		fmt.Println("\033[1;32mNote updated successfully!\033[0m")
		printNoteText(msg.Note)
	default:
		fmt.Printf("%+v\n", msg)
	}
}

func printError(err error, format string) {
	if format == "json" {
		resp := map[string]string{
			"status": "error",
			"error":  err.Error(),
		}
		var connectErr *connect.Error
		if errors.As(err, &connectErr) {
			resp["code"] = connectErr.Code().String()
			resp["message"] = connectErr.Message()
		}
		data, _ := json.MarshalIndent(resp, "", "  ")
		fmt.Println(string(data))
		return
	}

	var connectErr *connect.Error
	if errors.As(err, &connectErr) {
		fmt.Fprintf(os.Stderr, "\033[1;31mError (%s):\033[0m %s\n", connectErr.Code().String(), connectErr.Message())
		return
	}
	fmt.Fprintf(os.Stderr, "\033[1;31mError:\033[0m %v\n", err)
}
