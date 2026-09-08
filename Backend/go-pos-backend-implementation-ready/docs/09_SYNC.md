# Offline Synchronization

## Pull

Client sends:

```text
GET /v1/sync/pull?cursor=<sequence>&limit=<n>
```

Server returns changes ordered by ascending `change_seq`.

## Cursor

A cursor represents a server sequence, not a timestamp.

If a cursor is older than the retention window, return a dedicated `cursor_expired` error and require a controlled full resynchronization.

## Push

Each client command has a stable `command_id`.

The server records command processing so retries are replay-safe.

## Conflict policy

Do not invent one global conflict algorithm.

Define policy per entity:

- append-only: reject conflicting mutation;
- catalog: last-writer or server-authoritative depending on field;
- inventory: transactional server authority;
- customer edits: explicit field-level or revision conflict;
- sales: immutable after finalization except controlled reversal.

## Requirements

- monotonic ordering;
- bounded page size;
- deterministic replay;
- tenant-scoped command identity;
- transactionally consistent command application;
- observability of rejected/conflicting commands.
