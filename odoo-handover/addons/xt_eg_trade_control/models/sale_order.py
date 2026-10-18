from odoo import _, api, fields, models
from odoo.exceptions import UserError


class SaleOrder(models.Model):
    _inherit = "sale.order"

    eg_trade_sale_mode = fields.Selection(
        [("quick", "Quick Sale"), ("standard", "Standard Sale")],
        string="Sale Screen Mode",
        default="quick",
    )
    eg_trade_branch = fields.Char(string="Branch")
    eg_trade_warehouse_id = fields.Many2one("stock.warehouse", string="Operating Warehouse")
    eg_trade_eta_state = fields.Selection(
        [
            ("draft", "Draft"),
            ("ready", "Ready"),
            ("sent", "Sent"),
            ("accepted", "Accepted"),
            ("rejected", "Rejected"),
            ("cancelled", "Cancelled"),
            ("retry", "Retry Needed"),
        ],
        string="Egypt Tax Status",
        default="draft",
        copy=False,
    )
    eg_trade_eta_document_id = fields.Many2one("eg.trade.eta.document", string="Tax Follow-up", copy=False)
    eg_trade_payment_hint = fields.Selection(
        [("cash", "Cash"), ("bank", "Bank"), ("wallet", "Wallet"), ("credit", "Credit")],
        string="Expected Collection",
        default="cash",
    )
    eg_trade_simple_note = fields.Char(string="Simple Note")
    eg_trade_low_stock_warning = fields.Boolean(compute="_compute_eg_trade_warnings")
    eg_trade_credit_warning = fields.Boolean(compute="_compute_eg_trade_warnings")

    @api.depends("order_line.product_uom_qty", "partner_id", "amount_total", "order_line.product_id")
    def _compute_eg_trade_warnings(self):
        for order in self:
            credit_limit = order.partner_id.eg_trade_credit_limit
            order.eg_trade_credit_warning = bool(credit_limit and order.amount_total > credit_limit)
            order.eg_trade_low_stock_warning = any(
                line.product_id.type != "service" and line.product_id.qty_available < line.product_uom_qty
                for line in order.order_line
                if line.product_id
            )

    def action_eg_trade_prepare_delivery_invoice(self):
        for order in self:
            if order.eg_trade_credit_warning:
                raise UserError(_("Customer credit limit needs manager review before continuing."))
            if order.eg_trade_low_stock_warning:
                raise UserError(_("Some products do not have enough available quantity."))
            if order.state in ("draft", "sent"):
                order.action_confirm()
            if order.invoice_status == "to invoice":
                invoices = order._create_invoices()
                invoices.action_eg_trade_prepare_eta()
        return True

    def action_eg_trade_prepare_tax_followup(self):
        for order in self:
            document = order.eg_trade_eta_document_id or self.env["eg.trade.eta.document"].create({
                "document_type": "invoice",
                "source_model": "sale.order",
                "source_res_id": order.id,
                "sale_order_id": order.id,
                "partner_id": order.partner_id.id,
                "amount_total": order.amount_total,
                "company_id": order.company_id.id,
            })
            order.eg_trade_eta_document_id = document.id
            result = document.action_check_ready()
            order.eg_trade_eta_state = document.state
            if isinstance(result, dict):
                return result
        return True

    def action_eg_trade_open_quick_sale(self):
        self.ensure_one()
        return {
            "type": "ir.actions.act_window",
            "name": _("Easy Sale"),
            "res_model": "sale.order",
            "res_id": self.id,
            "view_mode": "form",
            "target": "current",
            "context": {"form_view_ref": "xt_eg_trade_control.view_sale_order_form_easy"},
        }
