from datetime import timedelta

from odoo import api, fields, models


class EgTradeDashboard(models.Model):
    _name = "eg.trade.dashboard"
    _description = "Trade Control Dashboard"

    name = fields.Char(default="Trade Control Dashboard", required=True)
    company_id = fields.Many2one("res.company", default=lambda self: self.env.company, required=True)

    def _period_start(self):
        today = fields.Date.context_today(self)
        return today, today.replace(day=1)

    @api.model
    def get_dashboard_data(self):
        today, month_start = self._period_start()
        company_domain = [("company_id", "=", self.env.company.id)]
        SaleOrder = self.env["sale.order"]
        Move = self.env["account.move"]
        Picking = self.env["stock.picking"]
        Product = self.env["product.template"]
        Eta = self.env["eg.trade.eta.document"]

        today_sales = sum(SaleOrder.search(company_domain + [("date_order", ">=", today), ("state", "in", ["sale", "done"])]).mapped("amount_total"))
        month_sales = sum(SaleOrder.search(company_domain + [("date_order", ">=", month_start), ("state", "in", ["sale", "done"])]).mapped("amount_total"))
        today_collected = sum(
            Move.search(
                company_domain
                + [
                    ("move_type", "in", ["out_invoice", "out_refund"]),
                    ("state", "=", "posted"),
                    ("invoice_date", "=", today),
                    ("payment_state", "in", ["paid", "in_payment"]),
                ]
            ).mapped("amount_total_signed")
        )
        eta_counts = {
            state: Eta.search_count(company_domain + [("state", "=", state)])
            for state in ["ready", "sent", "accepted", "rejected", "retry"]
        }
        reorder_products = Product.search([("eg_trade_reorder_min", ">", 0)])
        low_stock = len(reorder_products.filtered(lambda product: product.qty_available <= product.eg_trade_reorder_min))
        return {
            "company_name": self.env.company.name,
            "currency_symbol": self.env.company.currency_id.symbol or "",
            "currency_position": self.env.company.currency_id.position or "after",
            "today_sales": today_sales,
            "month_sales": month_sales,
            "today_collected": abs(today_collected),
            "pending_deliveries": Picking.search_count(company_domain + [("state", "not in", ["done", "cancel"]), ("picking_type_id.code", "=", "outgoing")]),
            "open_quotations": SaleOrder.search_count(company_domain + [("state", "in", ["draft", "sent"])]),
            "unpaid_invoices": Move.search_count(company_domain + [("move_type", "=", "out_invoice"), ("state", "=", "posted"), ("payment_state", "not in", ["paid", "reversed"])]),
            "low_stock": low_stock,
            "eta": eta_counts,
            "readiness_score": self._readiness_score(eta_counts, low_stock),
            "sales_series": self._sales_series(month_start),
        }

    def _readiness_score(self, eta_counts, low_stock):
        penalties = min(eta_counts.get("rejected", 0) * 8 + eta_counts.get("retry", 0) * 5 + low_stock * 2, 80)
        return max(20, 100 - penalties)

    def _sales_series(self, month_start):
        SaleOrder = self.env["sale.order"]
        series = []
        for index in range(6):
            day = month_start + timedelta(days=index * 5)
            end = day + timedelta(days=4)
            total = sum(
                SaleOrder.search([
                    ("company_id", "=", self.env.company.id),
                    ("date_order", ">=", day),
                    ("date_order", "<=", end),
                    ("state", "in", ["sale", "done"]),
                ]).mapped("amount_total")
            )
            series.append({"period": day.strftime("%d/%m"), "sales": total})
        return series
