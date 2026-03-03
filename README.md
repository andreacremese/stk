# stk

Local CLI for capturing short follow-up notes while pairing with AI agents.
Keep your coding flow by parking topics on a per-repo/branch stack and resuming them later.


**ALSO** when randomized jump back into wherever you were in the interacting with the agent.

## what is the need this fulfills?

AI assistants often surface multiple topics at once. You want to deep dive on one, but after a few exchanges you've lost track of the others.

**Example:** AI suggests debugging three areas: multithreading, logging, and client connections. You tackle connections first. After several messages, you need to scroll back to find what else was mentioned.

**Solution:** Park them on a stack. When done, `pop` and move to the next.

```sh
stk push "multithreading vs using runtime async"
stk push "logging"
stk push "clients dropping connections"
stk show
# 0  ...  clients dropping connections  ← tackle this first
# 1  ...  logging
# 2  ...  multithreading vs using runtime async
stk pop  # done with connections, move to logging
```

Each branch has its own stack, so you don't pollute unrelated work.


### what is another need?

I get randomized quite a bit, and I end up having to come back into a session after the agent is done, and I have to rekindle myself to "where was I? which were the threads I was following here?". Just going `stk show` helps me a lot.

## why not an MCP?

Refer to [this article](https://andreacremese.github.io/ai/mcp%20vs%20terminal%20for%20tool%20calls.html).
I prefer CLI in general - and in this case mostly as I can see myself using this on the command line when sharing a terminal.

## Usage

```
stk <command> [repo] [branch] [args]

Commands:
  push  [repo] [branch] "note"   Push a note onto the stack
  show  [repo] [branch]          List all notes (index 0 = top)
  pop   [repo] [branch]          Remove and print the top note
  clear [repo] [branch]          Remove all notes
  pluck [repo] [branch] <index>  Remove and print the note at index
```

`repo` and `branch` are optional. Omit them (or pass `_`) to resolve from the current git repository.

### Examples

```sh
# Auto-resolve repo and branch from current git repo
stk push "revisit error handling in store"
stk show
stk pop

# Explicit repo and branch
stk push myrepo main "check the time layout fix"
stk pluck myrepo main 1
```

## Install

Install directly to `$GOPATH/bin` (requires Go 1.21+):

```sh
go install github.com/andreacremese/stk@latest
```

Or build locally from source:

```sh
git clone https://github.com/andreacremese/stk
cd stk
make build    # builds ./stk
make install  # installs to $GOPATH/bin
```

## Data

Notes are stored in a local SQLite database at:

- **macOS**: `~/Library/Application Support/agent-stack/stacks.db`
- **Linux**: `~/.config/agent-stack/stacks.db`

Override the location with the `AGENT_STACK_DB_PATH` environment variable.

## Notes

- Note max length: 160 characters
- Stack order: index `0` is the top (newest entry)
- Stack key is `repo + branch` — each branch has its own independent stack

## How am I using this?

> these are notes I have in one of my agent instructions

I use `stk` with AI coding agents to avoid losing track of tangents without breaking flow.
The agent is instructed to use it proactively — no need to ask.

**When to push:****
- I say "park this", "note that", "add to stack", or "we'll come back to this"
- A tangent emerges that would break flow — the agent suggests parking it
- I explicitly ask to push something

**When to show / pop / pluck:**
- I say "what's parked", "what's on the stack", "resume a parked item"
- I say "let's tackle that" or "pick up where we left off"

**Agent invocation:**
Before using the tool, the agent runs `stk help` to get the current command syntax, so these instructions stay valid even as the tool evolves.
