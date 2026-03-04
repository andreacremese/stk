package main

// Command stk is a local CLI for managing short follow-up notes keyed
// by git repo+branch. It provides five subcommands: push, show, pop, clear,
// and pluck. Repo and branch arguments are optional; omitting them (or passing
// "_") resolves the values from the current git working directory.

import (
	"errors"
	"fmt"
	"os"
	"runtime/debug"
	"strconv"
	"time"

	"github.com/andreacremese/stk/internal/gitctx"
	"github.com/andreacremese/stk/internal/stack"
	"github.com/andreacremese/stk/internal/store"
)

func getVersion() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		if info.Main.Version != "" && info.Main.Version != "(devel)" {
			return info.Main.Version
		}
	}
	return "dev"
}

const usage = `Usage: stk <command> [repo] [branch] [args]

Commands:
  push  [repo] [branch] "note"   Push a note onto the stack
  show  [repo] [branch]          List all notes (index 0 = top)
  pop   [repo] [branch]          Remove and print the top note
  clear [repo] [branch]          Remove all notes
  pluck [repo] [branch] <index>  Remove and print the note at index

Repo and branch are optional. Omit them (or use "_") to resolve from the
current git repository.

Flags:
  --version   Print version and exit
`

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}
}

// run is the real entry-point, separated from main for testability.
func run(args []string) error {
	if len(args) == 0 || args[0] == "-h" || args[0] == "--help" || args[0] == "help" {
		fmt.Fprint(os.Stderr, usage)
		return nil
	}

	if args[0] == "--version" || args[0] == "-v" {
		fmt.Println(getVersion())
		return nil
	}

	cmd, rest := args[0], args[1:]

	dbPath, err := store.DefaultDBPath()
	if err != nil {
		return fmt.Errorf("resolving db path: %w", err)
	}

	db, err := store.Open(dbPath)
	if err != nil {
		return fmt.Errorf("opening store: %w", err)
	}
	defer db.Close()

	st := stack.New(db)

	switch cmd {
	case "push":
		return runPush(st, rest)
	case "show":
		return runShow(st, rest)
	case "pop":
		return runPop(st, rest)
	case "clear":
		return runClear(st, rest)
	case "pluck":
		return runPluck(st, rest)
	default:
		return fmt.Errorf("unknown command %q\n\n%s", cmd, usage)
	}
}

// ---- subcommand handlers ---------------------------------------------------

// runPush handles: push [repo] [branch] "note"
// The note is always the last argument.
func runPush(st *stack.Stack, args []string) error {
	switch len(args) {
	case 0:
		return fmt.Errorf("push requires a note\n\nUsage: stk push [repo] [branch] \"note\"")
	case 1, 2, 3:
	default:
		return fmt.Errorf("push: too many arguments\n\nUsage: stk push [repo] [branch] \"note\"")
	}

	note := args[len(args)-1]
	repo, branch := ctxFromLeading(args[:len(args)-1])

	ctx, err := resolveCtx(repo, branch)
	if err != nil {
		return err
	}

	item, err := st.Push(ctx.Repo, ctx.Branch, note)
	if err != nil {
		if errors.Is(err, stack.ErrNoteTooLong) {
			return fmt.Errorf("note is too long (%d chars, max %d)", len(note), stack.MaxNoteLen)
		}
		return err
	}

	fmt.Printf("pushed  %s  %s  %s\n", item.ID, fmtTime(item.CreatedAt), item.Note)
	return nil
}

// runShow handles: show [repo] [branch]
func runShow(st *stack.Stack, args []string) error {
	if len(args) > 2 {
		return fmt.Errorf("show: too many arguments\n\nUsage: stk show [repo] [branch]")
	}

	repo, branch := ctxFromLeading(args)
	ctx, err := resolveCtx(repo, branch)
	if err != nil {
		return err
	}

	items, err := st.Show(ctx.Repo, ctx.Branch)
	if err != nil {
		return err
	}

	if len(items) == 0 {
		fmt.Println("(empty)")
		return nil
	}

	for i, it := range items {
		fmt.Printf("%d  %s  %s  %s\n", i, it.ID, fmtTime(it.CreatedAt), it.Note)
	}
	return nil
}

// runPop handles: pop [repo] [branch]
func runPop(st *stack.Stack, args []string) error {
	if len(args) > 2 {
		return fmt.Errorf("pop: too many arguments\n\nUsage: stk pop [repo] [branch]")
	}

	repo, branch := ctxFromLeading(args)
	ctx, err := resolveCtx(repo, branch)
	if err != nil {
		return err
	}

	item, err := st.Pop(ctx.Repo, ctx.Branch)
	if err != nil {
		if errors.Is(err, stack.ErrEmptyStack) {
			return fmt.Errorf("stack is empty for %s/%s", ctx.Repo, ctx.Branch)
		}
		return err
	}

	fmt.Printf("popped  %s  %s  %s\n", item.ID, fmtTime(item.CreatedAt), item.Note)
	return nil
}

// runClear handles: clear [repo] [branch]
func runClear(st *stack.Stack, args []string) error {
	if len(args) > 2 {
		return fmt.Errorf("clear: too many arguments\n\nUsage: stk clear [repo] [branch]")
	}

	repo, branch := ctxFromLeading(args)
	ctx, err := resolveCtx(repo, branch)
	if err != nil {
		return err
	}

	if err := st.Clear(ctx.Repo, ctx.Branch); err != nil {
		return err
	}

	fmt.Printf("cleared %s/%s\n", ctx.Repo, ctx.Branch)
	return nil
}

// runPluck handles: pluck [repo] [branch] <index>
// The index is always the last argument.
func runPluck(st *stack.Stack, args []string) error {
	switch len(args) {
	case 0:
		return fmt.Errorf("pluck requires an index\n\nUsage: stk pluck [repo] [branch] <index>")
	case 1, 2, 3:
	default:
		return fmt.Errorf("pluck: too many arguments\n\nUsage: stk pluck [repo] [branch] <index>")
	}

	indexStr := args[len(args)-1]
	index, err := strconv.Atoi(indexStr)
	if err != nil {
		return fmt.Errorf("pluck: index must be a non-negative integer, got %q", indexStr)
	}

	repo, branch := ctxFromLeading(args[:len(args)-1])
	ctx, err := resolveCtx(repo, branch)
	if err != nil {
		return err
	}

	item, err := st.Pluck(ctx.Repo, ctx.Branch, index)
	if err != nil {
		if errors.Is(err, stack.ErrIndexOutOfRange) {
			return fmt.Errorf("index %d is out of range for %s/%s", index, ctx.Repo, ctx.Branch)
		}
		return err
	}

	fmt.Printf("plucked %s  %s  %s\n", item.ID, fmtTime(item.CreatedAt), item.Note)
	return nil
}

// ---- helpers ---------------------------------------------------------------

// ctxFromLeading reads [repo] and [branch] from the first two positions of
// args. Missing values are returned as empty strings, which gitctx.Resolve
// treats as "resolve from git".
func ctxFromLeading(args []string) (repo, branch string) {
	if len(args) >= 1 {
		repo = args[0]
	}
	if len(args) >= 2 {
		branch = args[1]
	}
	return repo, branch
}

// resolveCtx calls gitctx.Resolve with DefaultLookup and wraps ErrNotGitRepo
// with a user-friendly message.
func resolveCtx(repo, branch string) (gitctx.Ctx, error) {
	ctx, err := gitctx.Resolve(repo, branch, gitctx.DefaultLookup)
	if err != nil {
		if errors.Is(err, gitctx.ErrNotGitRepo) {
			return gitctx.Ctx{}, fmt.Errorf(
				"not inside a git repository; provide repo and branch explicitly",
			)
		}
		return gitctx.Ctx{}, err
	}
	return ctx, nil
}

// fmtTime formats a time.Time for display output.
func fmtTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}
