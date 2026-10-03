package opsconsole

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"m31labs.dev/fluffyui/runtime"
	"m31labs.dev/fluffyui/state"
	"m31labs.dev/fluffyui/widgets"

	"posgo-ops/internal/posgo"
)

// Status values mirrored from the wire package so the view can switch on them
// without importing wire details into every file.
const (
	statusActive    = posgo.StatusActive
	statusSuspended = posgo.StatusSuspended
	statusDisabled  = posgo.StatusDisabled
)

// statusPill is the display form of a tenant status.
func statusPill(status string) string { return posgo.StatusPill(status) }

// screen is which top-level view is showing. The console is small enough that a
// screen enum plus a tab strip beats a router.
type screen int

const (
	screenLogin screen = iota
	screenOverview
	screenTenants
	screenDetail
)

func (s screen) String() string {
	switch s {
	case screenLogin:
		return "Sign in"
	case screenOverview:
		return "Overview"
	case screenTenants:
		return "Tenants"
	case screenDetail:
		return "Tenant"
	default:
		return "?"
	}
}

// statusKind colours the status line, so an error can never be mistaken for a
// success at a glance.
type statusKind int

const (
	statusInfo statusKind = iota
	statusError
	statusSuccess
)

// Config is what main passes in: where the API lives and which credentials to
// pre-fill.
//
// Password is supplied out of band rather than through a visible input field on
// purpose. A terminal session is routinely recorded or shared over SSH, and a
// password rendered on screen would end up in that recording. StopReason is the
// justification attached to a platform stop; it is read from the environment for
// the same reason, so the incident being recorded is already in the operator's
// shell history rather than typed into a widget.
type Config struct {
	BaseURL     string
	TenantID    string
	Email       string
	Password    string
	StopReason  string
	BackupDir   string
	DeviceID    string
	DeviceName  string
	AccessToken string
}

// analyticsResult pairs a tenant's drill-down with the id it belongs to, so a
// response that lands after the operator moved on can be recognised and shown
// against the right tenant rather than the current selection.
type analyticsResult struct {
	TenantID  string
	Analytics posgo.Analytics
}

// Console is the root view. Widgets are built once and mutated in place as
// signals change, because rebuilding the tree on every keystroke would drop
// focus, scroll position and the operator's place in the table.
type Console struct {
	client *posgo.Client
	cfg    Config
	ctx    context.Context

	// mu guards rowIDs, which is written by the tenant fetch callback and read
	// by the open-detail handler. Everything else is a Signal, which is already
	// internally synchronised.
	mu     sync.RWMutex
	rowIDs []string

	filters  *state.Signal[posgo.Filters]
	message  *state.Signal[string]
	msgKind  *state.Signal[statusKind]
	selected *state.Signal[string]
	session  *state.Signal[posgo.Session]
	busy     *state.Signal[bool]
	// signedIn chooses between the login form and the authenticated panes. It is
	// a signal rather than a field on session so the view can subscribe to one
	// boolean instead of re-deriving "is there a token" on every paint.
	signedIn *state.Signal[bool]

	// confirming gates the destructive stop behind an explicit yes, and
	// confirmID pins the tenant that yes applies to, so moving the cursor in the
	// table cannot redirect a stop at a different merchant.
	confirming *state.Signal[bool]
	confirmID  *state.Signal[string]
	// lastBackup is the "path (size)" of the most recent export, shown on the
	// status line once the write completes.
	lastBackup *state.Signal[string]

	summary   *state.Resource[posgo.Summary]
	tenants   *state.Resource[posgo.TenantPage]
	analytics *state.Resource[analyticsResult]
	// stopped counts disabled tenants platform-wide. It is a separate request
	// rather than a tally of the visible page, because the page the operator is
	// looking at is filtered and paginated: counting it would put a number on
	// the overview that silently changes with the filters.
	stopped *state.Resource[int64]

	// Widgets kept for mutation.
	root         runtime.Widget
	tabs         *widgets.Tabs
	table        *widgets.Table
	spinner      *widgets.Spinner
	emailInput   *widgets.Input
	tenantIn     *widgets.Input
	tokenIn      *widgets.Input
	searchIn     *widgets.Input
	typeSel      *widgets.Select
	statusSel    *widgets.Select
	stopReasonIn *widgets.Input
	tiles        []*StatTile

	// marshal routes fetch-completion paints onto the UI thread, and
	// unsubPaints lets Run rebind them once the app's scheduler exists.
	marshal     state.Scheduler
	unsubPaints []func()
}

// bindScheduler points the repaint subscribers at the app's UI scheduler.
//
// This has to happen after the app exists because the console is assembled
// before it: a Resource notifies its subscribers on whichever goroutine the
// fetch resolved on, and mutating a widget tree from there races the render
// loop. Rebinding re-subscribes and repaints once, because
// SubscribeWithScheduler does not replay the current value the way a signal
// subscription does.
func (c *Console) bindScheduler(sched state.Scheduler) {
	if sched == nil {
		return
	}
	c.marshal = sched
	for _, unsub := range c.unsubPaints {
		unsub()
	}
	c.unsubPaints = c.subscribePaints(c.marshal)
	// Dispatched through the scheduler rather than called directly: a repaint
	// has to be serialised against the fetch goroutines the same way every other
	// repaint is, and on the app's scheduler this lands on the render thread
	// instead of on whichever goroutine happened to call bindScheduler.
	c.marshal.Schedule(func() {
		c.paintSummary()
		c.paintTenants()
	})
}

// New builds the console and returns the root widget to hand to the app.
func New(ctx context.Context, cfg Config) (*Console, runtime.Widget, error) {
	if strings.TrimSpace(cfg.BaseURL) == "" {
		return nil, nil, fmt.Errorf("opsconsole: base URL is required")
	}

	c := &Console{
		client:     posgo.NewClient(cfg.BaseURL, nil),
		cfg:        cfg,
		ctx:        ctx,
		filters:    state.NewSignal(posgo.Filters{Page: 1, Limit: 25}),
		message:    state.NewSignal(""),
		msgKind:    state.NewSignal(statusInfo),
		selected:   state.NewSignal(""),
		confirming: state.NewSignal(false),
		confirmID:  state.NewSignal(""),
		lastBackup: state.NewSignal(""),
		session:    state.NewSignal(posgo.Session{}),
		signedIn:   state.NewSignal(false),
		busy:       state.NewSignal(false),
		// Nothing is subscribed yet, and this scheduler drops anything that
		// arrives before bindScheduler. The alternative - a synchronous default
		// - means a fetch completing in the window between New and Run paints
		// the tiles on its own goroutine, racing the first real repaint. A
		// fetch that lands early is simply repainted by bindScheduler, which
		// reads the resource's current value anyway.
		marshal: state.SchedulerFunc(func(func()) {}),
	}

	c.buildResources()
	root := c.buildView()
	c.root = root

	if strings.TrimSpace(cfg.AccessToken) != "" {
		// A token was supplied: adopt it and let the first fetch prove whether
		// it is still good, rather than asking the operator to sign in again.
		c.client.SetToken(cfg.AccessToken)
		c.session.Update(func(s posgo.Session) posgo.Session {
			s.AccessToken = cfg.AccessToken
			s.Role = "saas_admin"
			return s
		})
		c.signedIn.Set(true)
		c.note(statusInfo, "using the supplied access token")
		c.refreshAll()
	}
	return c, root, nil
}

// buildResources creates the async fetchers. The tenant resource depends on the
// filter signal, so changing a filter refetches automatically and any response
// still in flight for the previous filter is discarded rather than painted
// over the new results.
//
// Every fetcher short-circuits while there is no access token. state.NewResource
// runs its fetcher once at construction, so without this guard the console fired
// three unauthenticated requests the moment it started and painted "api 401
// no_session" on the login form - an error the operator could do nothing about,
// for a request they had not made yet. The first real fetch is refreshAll,
// after sign-in.
func (c *Console) buildResources() {
	c.summary = state.NewResource(func() (posgo.Summary, error) {
		if !c.hasToken() {
			return posgo.Summary{}, nil
		}
		return c.client.Summary(c.ctx)
	})
	c.tenants = state.NewResource(func() (posgo.TenantPage, error) {
		if !c.hasToken() {
			return posgo.TenantPage{}, nil
		}
		return c.client.Tenants(c.ctx, c.filters.Get())
	}, c.filters)
	c.stopped = state.NewResource(func() (int64, error) {
		if !c.hasToken() {
			return 0, nil
		}
		// limit=1: only meta.total is wanted, not the rows.
		page, err := c.client.Tenants(c.ctx, posgo.Filters{Status: posgo.StatusDisabled, Limit: 1})
		if err != nil {
			return 0, err
		}
		return page.Total, nil
	})
	c.analytics = state.NewResource(func() (analyticsResult, error) {
		id := c.selected.Get()
		if id == "" || !c.hasToken() {
			return analyticsResult{}, nil
		}
		data, err := c.client.Analytics(c.ctx, id)
		if err != nil {
			return analyticsResult{}, err
		}
		return analyticsResult{TenantID: id, Analytics: data}, nil
	}, c.selected)
}

// hasToken reports whether there is a session to authenticate with.
func (c *Console) hasToken() bool {
	return strings.TrimSpace(c.client.Token()) != ""
}

// rowID returns the tenant id behind the table's selected row.
func (c *Console) rowID() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	idx := c.table.SelectedIndex()
	if idx < 0 || idx >= len(c.rowIDs) {
		return ""
	}
	return c.rowIDs[idx]
}

// setRows replaces the table contents and the id index that goes with it. The
// two are written together so a row can never be read back with someone else's
// id attached.
func (c *Console) setRows(ids []string, rows [][]string) {
	c.mu.Lock()
	c.rowIDs = ids
	c.mu.Unlock()
	c.table.SetRows(rows)
}

// note sets the status line.
func (c *Console) note(kind statusKind, format string, args ...any) {
	c.msgKind.Set(kind)
	c.message.Set(fmt.Sprintf(format, args...))
}

// handleAPIError turns a client error into a status line, sending an expired
// session back to the login screen instead of leaving a dead console up.
func (c *Console) handleAPIError(what string, err error) {
	var apiErr *posgo.APIError
	if errors.As(err, &apiErr) && apiErr.IsAuthFailure() {
		c.cancelConfirm()
		c.note(statusError, "%s: session expired, sign in again", what)
		c.client.SetToken("")
		c.session.Set(posgo.Session{})
		c.selected.Set("")
		c.signedIn.Set(false)
		return
	}
	c.note(statusError, "%s: %v", what, err)
}

// dispose releases the repaint subscriptions and the fetches' timers. It runs on
// shutdown so a fetch completing during teardown cannot repaint a dead tree.
func (c *Console) dispose() {
	for _, unsub := range c.unsubPaints {
		unsub()
	}
	c.unsubPaints = nil
	c.summary.Dispose()
	c.tenants.Dispose()
	c.analytics.Dispose()
	c.stopped.Dispose()
}
