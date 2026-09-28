# posgo-ops

The POS.Go control plane as a terminal application: tenant list, per-tenant
drill-down, and the activate / suspend / stop / backup actions, built with
[FluffyUI](https://pkg.go.dev/m31labs.dev/fluffyui).

It talks to the same `/v1/saas` and `/v1/platform` endpoints as the React admin
SPA, so an operator on the VPS can inspect and act on a merchant over SSH
without a browser.

## Why a separate Go module

This directory is its own module (`module posgo-ops`). The API server's
`go.mod` is deliberately left alone: FluffyUI is a terminal UI toolkit with no
place in an HTTP service, and adding it to the API's module would pull the whole
dependency graph into every build and every `go mod verify` the backend runs.

## Build and test

```bash
make check    # gofmt-check, go vet, go test
make race     # the suite under the race detector
make build    # -> bin/posgo-ops
```

## Run

Credentials come from the environment, never from flags, so they stay out of the
shell history and out of `ps` output on a shared box.

```bash
export POSGO_OPS_PASSWORD='…'
export POSGO_OPS_TENANT_ID='<the SaaS platform tenant uuid>'
make run                                   # local backend on :8080

# or against production
posgo-ops --api https://api.xamltech.com
```

| Variable | Meaning |
|---|---|
| `POSGO_OPS_API` | Base URL. Default `https://api.xamltech.com`. |
| `POSGO_OPS_PASSWORD` | SaaS staff password. |
| `POSGO_OPS_TENANT_ID` | The **SaaS platform** tenant id, required for password sign-in. |
| `POSGO_OPS_TOKEN` | An access token; skips the sign-in form entirely. |
| `POSGO_OPS_STOP_REASON` | Default justification recorded with a tenant stop. |
| `POSGO_OPS_BACKUP_DIR` | Where tenant exports are written. Default `$TMPDIR/posgo-backups`. |
| `POSGO_OPS_MCP` | Unix socket path for agent control. Empty disables it. |

The account needs the exact `saas_admin` role: the platform routes reject
anything else, and the console says so at sign-in rather than letting every
panel fail with a 403.

## What it does

- **Overview** — platform totals (tenants, users, sales, revenue) plus the
  per-business-type breakdown, and a **stopped** count of tenants currently
  locked out. That tile is fetched as its own query rather than tallied off the
  visible page, so the number does not change when the operator filters.
- **Tenants** — the full list, filterable by free text, business type and
  status, with a clamped pager.
- **Tenant detail** — plan, subdomain, subscription, join and trial dates, today's
  revenue and basket average, the 7-day trend, top products and recent sales.
- **Actions** — activate, suspend, stop and backup the selected tenant.

### Stop is gated

Stopping a tenant sets `tenants.status = 'disabled'`, which blocks the merchant
from signing in at all until an operator activates them again. So the stop button
only *arms* a confirmation strip: it replaces the action bar, names the tenant it
is about to stop, and takes a reason. Nothing is sent until **Confirm stop**.

The confirmation pins the tenant it was armed for. Moving the cursor in the table
while the strip is up does not redirect the stop.

### Backups go to a file

An export is the whole tenant, so it is written to disk (`0600`, in
`POSGO_OPS_BACKUP_DIR`) and the status line reports the path and size. Printing it
to the terminal would scroll the operator's session log full of customer rows.

## Agent control

`--mcp /tmp/posgo-ops.sock` opens a unix socket so an AI agent can read the screen
and drive the same controls. It is opt-in and announced on startup, because the
console is a privileged view of every merchant on the platform and opening a
control socket should be a decision an operator sees themselves.

## Notes on the code

- **Reactivity.** Fetches live in `state.Resource`s, and the view is built once
  and mutated in place. `Switcher` (`switcher.go`) exists because the framework
  has no reactive conditional: `fluffy.Conditional` picks its child when it is
  constructed, so it cannot switch. It mounts and exposes only the visible child,
  so a hidden confirmation strip is neither focusable nor mounted.
- **Threading.** A `Resource` notifies subscribers on whichever goroutine the
  fetch resolved on. `Run` therefore hands the console the app's scheduler
  (`bindScheduler`) so repaints happen on the render thread. The tests install a
  fake one they pump by hand, which is why the suite is clean under `-race`.
- **Styling.** The FSS stylesheet is built in Go (`styles.go`) rather than parsed
  from a `.fss` file, so the console stays a single binary with no sidecar to go
  missing on a VPS.
