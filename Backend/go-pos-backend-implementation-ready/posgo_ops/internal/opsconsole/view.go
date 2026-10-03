package opsconsole

import (
	"fmt"
	"strings"
	"time"

	"m31labs.dev/fluffyui/fluffy"
	"m31labs.dev/fluffyui/runtime"
	"m31labs.dev/fluffyui/state"
	"m31labs.dev/fluffyui/widgets"

	"posgo-ops/internal/posgo"
)

// tenantColumns are the tenant table's headings and widths. They are fixed so
// the grid stays aligned no matter how long a name or slug is.
var tenantColumns = []widgets.TableColumn{
	{Title: "Tenant", Width: 26},
	{Title: "Slug", Width: 24},
	{Title: "Type", Width: 14},
	{Title: "Status", Width: 11},
	{Title: "Plan", Width: 10},
	{Title: "Users", Width: 6},
	{Title: "Revenue", Width: 14},
}

// Tab indices within the authenticated strip, which is what the nav bar
// switches between. The strip doubles as the screen switcher, so these are the
// single place the layout and the navigation agree on ordering.
const (
	tabOverview = iota
	tabTenants
	tabDetail
)

// buildView assembles the whole console. Panels are built once and then
// repainted in place: the reactive leaves redraw from their signals, the login
// and authenticated bodies are chosen by a Switcher, and the table is rewritten
// by its subscriber.
func (c *Console) buildView() runtime.Widget {
	c.buildInputs()

	c.spinner = widgets.NewSpinner()
	c.tiles = []*StatTile{
		NewStatTile("Tenants", "—", "merchants on the platform", false),
		NewStatTile("Users", "—", "staff accounts across merchants", false),
		NewStatTile("Sales", "—", "completed sales", false),
		NewStatTile("Revenue", "—", "lifetime merchant revenue", false),
		NewStatTile("Stopped", "—", "tenants currently disabled", false),
	}
	c.table = widgets.NewTable(tenantColumns...)
	c.table.SetRows([][]string{{"loading…"}})

	// The three authenticated panes are built once and never reassigned.
	//
	// They used to be swapped into Tabs.Tabs on sign-in, which does not work:
	// widgets.Tabs only binds and mounts the selected content when the selected
	// index changes, so replacing the slice while the index stayed at zero left
	// the new panels unbound and never laid out. They painted on top of the nav
	// bar at the root origin. Choosing between two already-built bodies with a
	// Switcher keeps the tab list immutable, so the framework's own tab
	// switching does the binding.
	c.tabs = widgets.NewTabs(
		widgets.Tab{Title: "Overview", Content: c.overviewPanel()},
		widgets.Tab{Title: "Tenants", Content: c.tenantPanel()},
		widgets.Tab{Title: "Tenant detail", Content: c.detailPanel()},
	)
	c.tabs.SetLabel("Control plane views")

	return fluffy.VStack(
		c.navBar(),
		fluffy.Divider(),
		NewSwitcher(c.signedIn, c.tabs, c.loginBody()),
		fluffy.Divider(),
		c.statusBar(),
	)
}

// subscribePaints registers the fetch-completion repaints on the given scheduler
// and returns their unsubscribe handles. Only bindScheduler calls it: a Resource
// notifies subscribers on whichever goroutine the fetch resolved on, so these
// must never be registered before a scheduler is known.
func (c *Console) subscribePaints(sched state.Scheduler) []func() {
	return []func(){
		c.tenants.SubscribeWithScheduler(sched, c.paintTenants),
		c.summary.SubscribeWithScheduler(sched, c.paintSummary),
		c.stopped.SubscribeWithScheduler(sched, c.paintSummary),
	}
}

// buildInputs creates the filter and login widgets and wires their handlers.
func (c *Console) buildInputs() {
	c.emailInput = widgets.NewInput()
	c.emailInput.SetLabel("SaaS staff email")
	c.emailInput.SetPlaceholder("admin@posgo.saas")
	c.emailInput.SetText(c.cfg.Email)

	c.tenantIn = widgets.NewInput()
	c.tenantIn.SetLabel("Platform tenant id")
	c.tenantIn.SetPlaceholder("SaaS platform tenant UUID")
	c.tenantIn.SetText(c.cfg.TenantID)

	c.tokenIn = widgets.NewInput()
	c.tokenIn.SetLabel("Access token")
	c.tokenIn.SetPlaceholder("optional: skips the form")

	c.searchIn = widgets.NewInput()
	c.searchIn.SetLabel("Search tenants")
	c.searchIn.SetPlaceholder("search name, slug or email")
	c.searchIn.SetOnChange(func(text string) {
		c.updateFilters(func(f *posgo.Filters) { f.Search = text })
	})

	c.typeSel = widgets.NewSelect(optionsFor(posgo.BusinessTypeOptions)...)
	c.typeSel.SetLabel("Business type")
	c.typeSel.SetOnChange(func(opt widgets.SelectOption) {
		value, _ := opt.Value.(string)
		c.updateFilters(func(f *posgo.Filters) { f.BusinessType = value })
	})

	c.statusSel = widgets.NewSelect(optionsFor(posgo.StatusOptions)...)
	c.statusSel.SetLabel("Status")
	c.statusSel.SetOnChange(func(opt widgets.SelectOption) {
		value, _ := opt.Value.(string)
		c.updateFilters(func(f *posgo.Filters) { f.Status = value })
	})

	// The reason is part of the audit trail for a stop, so it is captured at the
	// moment of the stop rather than at startup.
	c.stopReasonIn = widgets.NewInput()
	c.stopReasonIn.SetLabel("Reason recorded with a stop")
	c.stopReasonIn.SetPlaceholder("e.g. chargeback, unpaid invoice")
	c.stopReasonIn.SetText(c.cfg.StopReason)
}

// confirmStrip is the inline two-step confirmation for the destructive stop.
//
// It replaces the action buttons rather than floating over them, so the stop is
// never one stray keystroke away and the operator sees exactly what "yes" does
// on the same line they are already looking at.
func (c *Console) confirmStrip() runtime.Widget {
	return fluffy.VStack(
		fluffy.ReactiveText(func() string {
			if !c.confirming.Get() {
				return ""
			}
			return fmt.Sprintf("Stop %s? The merchant cannot sign in until an operator reactivates them.",
				c.nameFor(c.confirmID.Get()))
		}, c.confirming, c.confirmID),
		fluffy.HStack(
			c.stopReasonIn,
			fluffy.DangerButton("Confirm stop", c.confirmStop),
			fluffy.SecondaryButton("Cancel", c.cancelConfirm),
		),
	)
}

// optionsFor turns wire values into select options, giving the empty string a
// readable label so a filter reads "All" rather than a blank row.
func optionsFor(values []string) []widgets.SelectOption {
	labels := map[string]string{
		"":              "All",
		statusActive:    "Active",
		statusSuspended: "Suspended",
		statusDisabled:  "Stopped",
		"restaurant":    "Restaurant",
		"coffee_shop":   "Coffee shop",
		"grocery":       "Grocery",
		"book_store":    "Book store",
		"mobile_shop":   "Mobile shop",
		"computer_shop": "Computer shop",
	}
	out := make([]widgets.SelectOption, 0, len(values))
	for _, v := range values {
		label := labels[v]
		if label == "" {
			label = v
		}
		out = append(out, widgets.SelectOption{Label: label, Value: v})
	}
	return out
}

// navBar is the screen switcher plus the signed-in identity.
func (c *Console) navBar() runtime.Widget {
	return fluffy.HStack(
		fluffy.Bold("POS.Go control plane"),
		fluffy.Text("  "),
		fluffy.PrimaryButton("Overview", func() { c.showScreen(screenOverview) }),
		fluffy.PrimaryButton("Tenants", func() { c.showScreen(screenTenants) }),
		fluffy.SecondaryButton("Detail", func() { c.showScreen(screenDetail) }),
		fluffy.SecondaryButton("Refresh", c.refreshAll),
		fluffy.SecondaryButton("Sign out", c.signOut),
		hGap(1),
		fluffy.ReactiveText(func() string {
			s := c.session.Get()
			if s.DisplayName == "" {
				return ""
			}
			return fmt.Sprintf("%s (%s)", s.DisplayName, s.Role)
		}, c.session),
	)
}

// loginBody is the unauthenticated form.
func (c *Console) loginBody() runtime.Widget {
	return fluffy.NewPanel(fluffy.VStack(
		fluffy.Bold("Sign in"),
		fluffy.Text("SaaS staff credentials for the control plane."),
		vGap(1),
		fluffy.Text("Email"),
		c.emailInput,
		fluffy.Text("Platform tenant id"),
		c.tenantIn,
		fluffy.Text("Access token (optional, skips the form)"),
		c.tokenIn,
		vGap(1),
		fluffy.PrimaryButton("Sign in", c.signIn),
	))
}

// overviewPanel is the platform summary: headline tiles plus the per-business
// breakdown the SaaS team reviews each morning.
func (c *Console) overviewPanel() runtime.Widget {
	// Built as explicit flex children rather than a plain HStack so the tiles
	// can be separated: each tile is exactly as wide as its widest line, so
	// without a gap "Revenue" runs straight into the next caption.
	children := make([]fluffy.FlexChild, 0, len(c.tiles)*2)
	for i, tile := range c.tiles {
		if i > 0 {
			children = append(children, fluffy.FixedSpace(3))
		}
		children = append(children, fluffy.Fixed(tile))
	}

	breakdown := multilineText(func() string {
		res := c.summary.Get()
		if res.Error != nil {
			return fmt.Sprintf("business breakdown unavailable: %v", res.Error)
		}
		if len(res.Data.ByBusiness) == 0 && res.Loading {
			return "loading business breakdown…"
		}
		lines := []string{fmt.Sprintf("%-16s %8s %8s %14s", "BUSINESS", "TENANTS", "USERS", "REVENUE")}
		for _, row := range sortedBusiness(res.Data.ByBusiness) {
			lines = append(lines, fmt.Sprintf("%-16s %8d %8d %14s",
				truncate(row.BusinessType, 16), row.Tenants, row.Users,
				posgo.FormatMinor(row.RevenueMinor, res.Data.SupportedCurrency)))
		}
		return strings.Join(lines, "\n")
	}, c.summary)

	return fluffy.VStack(
		fluffy.HFlex(children...),
		vGap(1),
		panel("Business breakdown", breakdown),
	)
}

// tenantPanel is the filterable tenant table plus the platform actions.
func (c *Console) tenantPanel() runtime.Widget {
	return fluffy.VStack(
		fluffy.HStack(
			c.searchIn,
			c.typeSel,
			c.statusSel,
		),
		fluffy.ReactiveText(func() string {
			f := c.filters.Get()
			page := c.tenants.Get().Data
			limit := f.Limit
			if limit < 1 {
				limit = 25
			}
			return fmt.Sprintf("%d matches · page %d/%d",
				page.Total, f.Page, posgo.TotalPages(page.Total, limit))
		}, c.filters, c.tenants),
		panel("Tenants", c.table),
		fluffy.HStack(
			fluffy.SecondaryButton("Prev", func() { c.setPage(-1) }),
			fluffy.SecondaryButton("Next", func() { c.setPage(1) }),
			fluffy.SecondaryButton("Open selected", c.openSelected),
			hGap(1),
		),
		c.actionBar(),
	)
}

// actionBar renders either the action buttons or, while a destructive stop is
// armed, the confirmation strip in their place.
//
// Both branches are built once and the Switcher picks between them, because the
// strip has to appear the moment the operator presses Stop.
func (c *Console) actionBar() runtime.Widget {
	children := []runtime.Widget{fluffy.Bold("Actions on the selected tenant: ")}
	for _, action := range c.platformActions() {
		action := action
		label := action.verb + " selected"
		if action.destructive {
			children = append(children, fluffy.DangerButton(label, func() { c.run(action) }))
			continue
		}
		children = append(children, fluffy.SecondaryButton(label, func() { c.run(action) }))
	}
	return NewSwitcher(c.confirming, c.confirmStrip(), fluffy.HStack(children...))
}

// detailPanel is the per-tenant drill-down.
func (c *Console) detailPanel() runtime.Widget {
	return fluffy.VStack(
		fluffy.HStack(
			fluffy.SecondaryButton("Back to tenants", c.closeDetail),
			fluffy.ReactiveText(func() string {
				res := c.analytics.Get()
				if res.Data.TenantID != "" {
					return res.Data.Analytics.Tenant.Name
				}
				return c.selected.Get()
			}, c.analytics),
		),
		panel("Tenant detail", multilineText(func() string {
			res := c.analytics.Get()
			if res.Error != nil {
				return fmt.Sprintf("tenant detail unavailable: %v", res.Error)
			}
			if res.Data.TenantID == "" {
				if c.selected.Get() == "" {
					return "select a tenant in the Tenants tab, then Open selected"
				}
				return "loading tenant detail…"
			}
			return renderAnalytics(res.Data.Analytics)
		}, c.analytics, c.selected)),
	)
}

// renderAnalytics lays out the drill-down as fixed-width text. It is a pure
// function so the layout can be asserted in a test without a terminal.
func renderAnalytics(a posgo.Analytics) string {
	currency := a.Tenant.CurrencyCode
	var b strings.Builder

	row := func(label, value string) {
		fmt.Fprintf(&b, "%-16s %s\n", label, value)
	}
	row("Tenant", a.Tenant.Name)
	row("Slug", a.Tenant.Slug)
	row("ID", a.Tenant.ID)
	row("Status", statusPill(a.Tenant.Status))
	row("Plan", a.Tenant.Plan)
	if a.Tenant.Subdomain != "" {
		row("Subdomain", a.Tenant.Subdomain)
	}
	if a.Tenant.SubscriptionStatus != "" {
		row("Subscription", a.Tenant.SubscriptionStatus)
	}
	row("Joined", fmt.Sprintf("%s (%s, %s)",
		posgo.FormatDate(a.Tenant.CreatedAt),
		posgo.RelativeAge(a.Tenant.CreatedAt, nowFunc()),
		a.Tenant.DefaultLanguage))
	if a.Tenant.TrialEndsAt != "" {
		row("Trial ends", posgo.FormatDate(a.Tenant.TrialEndsAt))
	}

	b.WriteString("\n")
	row("Users", fmt.Sprintf("%d", a.Counts.Users))
	row("Products", fmt.Sprintf("%d", a.Counts.Products))
	row("Revenue today", posgo.FormatMinor(a.Today.RevenueMinor, currency))
	row("Sales today", fmt.Sprintf("%d (avg %s)",
		a.Today.SalesCount, posgo.FormatMinor(a.Today.AvgSaleMinor, currency)))
	row("Items sold", fmt.Sprintf("%d", a.Today.ItemsSold))

	if len(a.RevenueTrend) > 0 {
		b.WriteString("\n7-day revenue\n")
		for _, point := range a.RevenueTrend {
			fmt.Fprintf(&b, "  %-12s %14s\n", point.Day, posgo.FormatMinor(point.RevenueMinor, currency))
		}
	}

	if len(a.TopProducts) > 0 {
		b.WriteString("\nTop products\n")
		for _, product := range a.TopProducts {
			fmt.Fprintf(&b, "  %-24s %4d x %12s\n",
				truncate(product.ProductName, 24), product.Quantity,
				posgo.FormatMinor(product.RevenueMinor, currency))
		}
	}

	if len(a.RecentSales) > 0 {
		b.WriteString("\nRecent sales\n")
		for _, sale := range a.RecentSales {
			fmt.Fprintf(&b, "  %-10s %12s %-8s %s\n",
				truncate(sale.Status, 10),
				posgo.FormatMinor(sale.TotalMinor, firstNonEmpty(sale.Currency, currency)),
				truncate(sale.PaymentMethod, 8),
				truncate(sale.Cashier, 16))
		}
	}

	return b.String()
}

// firstNonEmpty returns the first non-blank value. It is used where the API may
// omit a per-row currency and the tenant's own is the right fallback.
func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// statusBar is the single-line feedback area, prefixed by outcome.
func (c *Console) statusBar() runtime.Widget {
	return fluffy.HStack(
		c.spinner,
		fluffy.ReactiveText(func() string {
			message := c.message.Get()
			if message == "" {
				return ""
			}
			switch c.msgKind.Get() {
			case statusError:
				return "error: " + message
			case statusSuccess:
				return "ok: " + message
			default:
				return "• " + message
			}
		}, c.message, c.msgKind),
	)
}

// panel wraps content in a titled, bordered card.
func panel(title string, body runtime.Widget) runtime.Widget {
	return widgets.NewCard(title, body)
}

// nowFunc is indirected so the analytics layout is testable with a fixed clock.
var nowFunc = func() time.Time { return time.Now() }
