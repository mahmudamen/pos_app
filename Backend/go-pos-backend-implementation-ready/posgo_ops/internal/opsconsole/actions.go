package opsconsole

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"posgo-ops/internal/posgo"
)

// humanBytes renders a backup size for the status line.
func humanBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for m := n / unit; m >= unit; m /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(n)/float64(div), "KMGT"[exp])
}

// signIn authenticates the console. A pasted access token short-circuits the
// form, which is how an operator reuses a session exported from the web admin
// instead of retyping credentials onto a shared screen.
func (c *Console) signIn() {
	if token := strings.TrimSpace(c.tokenIn.Text()); token != "" {
		c.client.SetToken(token)
		c.session.Update(func(s posgo.Session) posgo.Session {
			s.AccessToken = token
			s.Role = "saas_admin"
			return s
		})
		c.afterSignIn("using supplied access token")
		return
	}

	email := strings.TrimSpace(c.emailInput.Text())
	tenantID := strings.TrimSpace(c.tenantIn.Text())
	switch {
	case email == "":
		c.note(statusError, "email is required")
		return
	case tenantID == "":
		c.note(statusError, "platform tenant id is required")
		return
	}

	c.busy.Set(true)
	session, err := c.client.Login(c.ctx, email, c.cfg.Password, tenantID, c.cfg.DeviceID, c.cfg.DeviceName)
	c.busy.Set(false)
	if err != nil {
		c.handleAPIError("sign in", err)
		return
	}
	c.session.Set(session)
	if !session.IsAdmin() {
		// The server gates these routes on the exact saas_admin role, so warn
		// now rather than letting every panel fail with a 403 later.
		c.note(statusError, "signed in as %q: the control plane needs the saas_admin role", session.Role)
	}
	c.afterSignIn(fmt.Sprintf("signed in as %s", session.DisplayName))
}

// afterSignIn reveals the authenticated panes and primes every panel.
func (c *Console) afterSignIn(note string) {
	c.signedIn.Set(true)
	c.note(statusSuccess, "%s", note)
	c.refreshAll()
}

// signOut clears the session and returns to the login form.
//
// It also disarms a pending stop. The confirmation pins its tenant id, so
// leaving it armed across a sign-out would re-arm a destructive action against
// the *next* session, on a tenant picked before this one.
func (c *Console) signOut() {
	c.cancelConfirm()
	c.client.SetToken("")
	c.session.Set(posgo.Session{})
	c.selected.Set("")
	c.signedIn.Set(false)
	c.note(statusInfo, "signed out")
}

// updateFilters mutates the filter set and resets to page one. Setting the
// signal is what triggers the tenant refetch, so no explicit reload is needed.
func (c *Console) updateFilters(mutate func(*posgo.Filters)) {
	current := c.filters.Get()
	mutate(&current)
	current.Page = 1
	c.filters.Set(current)
}

// setPage moves the pager, clamping to the page range the API reported so a
// filter change that shrinks the result set cannot strand the operator on an
// empty page 40.
func (c *Console) setPage(delta int) {
	current := c.filters.Get()
	limit := current.Limit
	if limit < 1 {
		limit = 25
	}
	pages := posgo.TotalPages(c.tenants.Get().Data.Total, limit)
	next := current.Page + delta
	if next < 1 {
		next = 1
	}
	if next > pages {
		next = pages
	}
	if next == current.Page {
		return
	}
	current.Page = next
	c.filters.Set(current)
}

// showScreen switches the visible tab. Detail is only reachable with a tenant
// selected, since its resource is keyed on the selection and would otherwise
// fetch nothing and look broken.
func (c *Console) showScreen(target screen) {
	if target == screenLogin {
		c.signOut()
		return
	}
	if c.session.Get().AccessToken == "" {
		c.note(statusError, "sign in first")
		return
	}
	if target == screenDetail && c.selected.Get() == "" {
		c.note(statusError, "select a tenant in the Tenants tab first")
		c.tabs.SetSelected(tabIndex(screenTenants))
		c.tabs.Invalidate()
		return
	}
	c.tabs.SetSelected(tabIndex(target))
	c.tabs.Invalidate()
}

// tabIndex maps a screen onto its position in the authenticated tab strip.
func tabIndex(target screen) int {
	switch target {
	case screenOverview:
		return tabOverview
	case screenDetail:
		return tabDetail
	default:
		return tabTenants
	}
}

// refreshAll re-fetches every panel, which is what an operator wants after a
// platform action has changed something.
func (c *Console) refreshAll() {
	c.summary.Refetch()
	c.tenants.Refetch()
	c.stopped.Refetch()
	if id := c.selected.Get(); id != "" {
		c.analytics.Refetch()
	}
}

// openSelected drills into the tenant under the cursor.
func (c *Console) openSelected() {
	id := c.rowID()
	if id == "" {
		c.note(statusError, "select a tenant first")
		return
	}
	c.selected.Set(id)
	c.showScreen(screenDetail)
}

// closeDetail returns to the tenant list, dropping the selection so the
// analytics resource stops holding a stale tenant.
func (c *Console) closeDetail() {
	c.selected.Set("")
	c.tabs.SetSelected(tabIndex(screenTenants))
	c.tabs.Invalidate()
}

// platformAction is one operator-triggered control-plane operation.
type platformAction struct {
	verb string
	// destructive marks actions that lock a merchant out, which get the danger
	// button treatment and a confirmation step.
	destructive bool
	run         func(id, reason string) error
}

// platformActions are the operator-triggered control-plane operations, in the
// order they are offered.
func (c *Console) platformActions() []platformAction {
	return []platformAction{
		{verb: "Activate", run: func(id, _ string) error { return c.client.Activate(c.ctx, id) }},
		{verb: "Suspend", run: func(id, _ string) error { return c.client.Suspend(c.ctx, id) }},
		// Stop is destructive: it sets tenants.status='disabled' and the
		// merchant cannot sign in again until an operator activates them. It
		// therefore goes through requestAction, which asks first.
		{verb: "Stop", destructive: true, run: func(id, reason string) error {
			return c.client.Stop(c.ctx, id, reason)
		}},
		{verb: "Backup", run: c.writeBackup},
	}
}

// writeBackup streams a tenant export to disk and reports where it landed.
//
// The bytes are deliberately written to a file rather than echoed to the
// terminal: an export is the whole tenant, which would both flood the screen
// and scroll the operator's session log full of customer data.
func (c *Console) writeBackup(id, _ string) error {
	data, err := c.client.Backup(c.ctx, id)
	if err != nil {
		return err
	}
	dir := c.cfg.BackupDir
	if dir == "" {
		dir = filepath.Join(os.TempDir(), "posgo-backups")
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create backup directory: %w", err)
	}
	name := fmt.Sprintf("%s-%s.json", time.Now().UTC().Format("20060102T150405Z"), id)
	path := filepath.Join(dir, name)
	// 0600: the export contains every row of the tenant, so it is not readable
	// by other users on a shared box.
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		return fmt.Errorf("write backup: %w", err)
	}
	c.lastBackup.Set(fmt.Sprintf("%s (%s)", path, humanBytes(int64(len(data)))))
	return nil
}

// run performs a platform action against the tenant under the cursor.
func (c *Console) run(action platformAction) {
	id := c.rowID()
	if id == "" {
		c.note(statusError, "select a tenant first")
		return
	}
	if action.destructive {
		// Arm the confirmation instead of acting on it: the tenant is pinned
		// now so the yes cannot land on whatever row the cursor moved to.
		c.confirmID.Set(id)
		c.confirming.Set(true)
		c.note(statusInfo, "stop %s? it locks the merchant out until reactivated", c.nameFor(id))
		return
	}
	c.perform(action, id)
}

// confirmStop carries out the stop that requestAction armed.
func (c *Console) confirmStop() {
	id := c.confirmID.Get()
	action := c.stopAction()
	if id == "" {
		c.cancelConfirm()
		return
	}
	c.perform(action, id)
	c.cancelConfirm()
}

// cancelConfirm stands the confirmation down without acting.
func (c *Console) cancelConfirm() {
	c.confirming.Set(false)
	c.confirmID.Set("")
}

// perform runs the action and reports the outcome on the status line.
func (c *Console) perform(action platformAction, id string) {
	reason := strings.TrimSpace(c.stopReasonIn.Text())
	if reason == "" {
		reason = c.cfg.StopReason
	}
	c.busy.Set(true)
	err := action.run(id, reason)
	c.busy.Set(false)
	if err != nil {
		c.handleAPIError(strings.ToLower(action.verb), err)
		return
	}
	if backup := c.lastBackup.Get(); action.verb == "Backup" && backup != "" {
		c.note(statusSuccess, "backup written to %s", backup)
	} else {
		c.note(statusSuccess, "%s ok: %s", strings.ToLower(action.verb), c.nameFor(id))
	}
	c.refreshAll()
}

// actionByVerb finds an action by its button label. The confirmation strip and
// the button both go through this, so they cannot disagree about what "yes" on
// the strip actually does.
func (c *Console) actionByVerb(verb string) (platformAction, bool) {
	for _, action := range c.platformActions() {
		if action.verb == verb {
			return action, true
		}
	}
	return platformAction{}, false
}

// stopAction returns the destructive action.
func (c *Console) stopAction() platformAction {
	for _, action := range c.platformActions() {
		if action.destructive {
			return action
		}
	}
	return platformAction{verb: "Stop"}
}

// nameFor is the tenant's display name if it is on the current page, so status
// messages read "stopped Demo Restaurant" rather than a UUID.
func (c *Console) nameFor(id string) string {
	for _, row := range c.tenants.Get().Data.Tenants {
		if row.ID == id {
			return row.Name
		}
	}
	return id
}
