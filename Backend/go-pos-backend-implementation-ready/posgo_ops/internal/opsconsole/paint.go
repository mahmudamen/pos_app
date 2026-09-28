package opsconsole

import (
	"fmt"
	"sort"

	"posgo-ops/internal/posgo"
)

// paintTenants repaints the table from the tenant resource. It runs on the
// resource's subscriber, which the framework invokes after the fetch lands, so
// it never touches the widget from the fetch goroutine itself.
func (c *Console) paintTenants() {
	res := c.tenants.Get()

	switch {
	case res.Error != nil:
		c.setRows(nil, [][]string{{"error: " + truncate(res.Error.Error(), 90)}})
		c.note(statusError, "tenants: %v", res.Error)
		return
	case res.Loading && len(res.Data.Tenants) == 0:
		c.setRows(nil, [][]string{{"loading…"}})
		return
	case len(res.Data.Tenants) == 0:
		// An empty result is a normal answer, not an error: the operator's
		// filter simply matched nothing.
		c.setRows(nil, [][]string{{"no tenants match this filter"}})
		return
	}

	currency := "EGP"
	ids := make([]string, 0, len(res.Data.Tenants))
	rows := make([][]string, 0, len(res.Data.Tenants))
	for _, tenant := range res.Data.Tenants {
		ids = append(ids, tenant.ID)
		rows = append(rows, []string{
			pad(tenant.Name, tenantColumns[0].Width),
			pad(tenant.Slug, tenantColumns[1].Width),
			pad(tenant.BusinessType, tenantColumns[2].Width),
			pad(statusPill(tenant.Status), tenantColumns[3].Width),
			pad(tenant.Plan, tenantColumns[4].Width),
			pad(fmt.Sprintf("%d", tenant.Users), tenantColumns[5].Width),
			pad(posgo.FormatMinor(tenant.RevenueMinor, currency), tenantColumns[6].Width),
		})
	}
	c.setRows(ids, rows)
}

// paintSummary repaints the overview tiles from the summary resource.
func (c *Console) paintSummary() {
	res := c.summary.Get()
	if res.Error != nil {
		for _, tile := range c.tiles {
			tile.SetValue("—")
		}
		return
	}
	data := res.Data

	c.tiles[0].SetValue(fmt.Sprintf("%d", data.TotalTenants))
	c.tiles[1].SetValue(posgo.FormatCount(int64(data.TotalUsers)))
	c.tiles[2].SetValue(posgo.FormatCount(data.TotalSales))
	c.tiles[3].SetValue(posgo.FormatMinor(data.RevenueMinor, data.SupportedCurrency))

	stopped := c.stopped.Get()
	switch {
	case stopped.Loading && stopped.Data == 0:
		c.tiles[4].SetValue("…")
	case stopped.Error != nil:
		// Never show a zero here: "0 stopped" reads as good news, and an
		// operator deciding whether to chase a locked-out merchant must not be
		// told there are none when the count simply failed.
		c.tiles[4].SetValue("—")
		c.tiles[4].SetUrgent(true)
	default:
		c.tiles[4].SetValue(fmt.Sprintf("%d", stopped.Data))
		c.tiles[4].SetUrgent(stopped.Data > 0)
	}
}

// sortedBusiness returns the breakdown rows ordered by tenant count, which is
// the order an operator scanning for the biggest movers wants. Equal counts fall
// back to the name so that two consecutive refreshes cannot reorder equal rows.
func sortedBusiness(rows []posgo.BusinessSlice) []posgo.BusinessSlice {
	out := append([]posgo.BusinessSlice(nil), rows...)
	sort.Slice(out, func(i, j int) bool {
		if out[i].Tenants != out[j].Tenants {
			return out[i].Tenants > out[j].Tenants
		}
		return out[i].BusinessType < out[j].BusinessType
	})
	return out
}
