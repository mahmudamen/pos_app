from odoo import _, api, fields, models
from odoo.exceptions import UserError


class EgTradeEtaDocument(models.Model):
    _name = "eg.trade.eta.document"
    _description = "Egypt Tax Document"
    _inherit = ["mail.thread", "mail.activity.mixin"]
    _order = "create_date desc"

    name = fields.Char(default="/", readonly=True, copy=False)
    company_id = fields.Many2one("res.company", default=lambda self: self.env.company, required=True)
    currency_id = fields.Many2one("res.currency", related="company_id.currency_id", readonly=True)
    document_type = fields.Selection(
        [
            ("invoice", "Invoice"),
            ("credit_note", "Credit Note"),
            ("debit_note", "Debit Note"),
            ("receipt", "Receipt"),
        ],
        default="invoice",
        required=True,
        tracking=True,
    )
    source_model = fields.Char(readonly=True)
    source_res_id = fields.Integer(readonly=True)
    move_id = fields.Many2one("account.move", string="Invoice", ondelete="set null")
    sale_order_id = fields.Many2one("sale.order", string="Sales Order", ondelete="set null")
    partner_id = fields.Many2one("res.partner", string="Customer", tracking=True)
    amount_total = fields.Monetary(currency_field="currency_id")
    eta_uuid = fields.Char(string="ETA UUID", copy=False, index=True)
    state = fields.Selection(
        [
            ("draft", "Draft"),
            ("ready", "Ready"),
            ("sent", "Sent"),
            ("accepted", "Accepted"),
            ("rejected", "Rejected"),
            ("cancelled", "Cancelled"),
            ("retry", "Retry Needed"),
        ],
        default="draft",
        tracking=True,
    )
    last_error = fields.Text(copy=False)
    retry_count = fields.Integer(readonly=True, copy=False)
    last_send_date = fields.Datetime(readonly=True, copy=False)
    responsible_id = fields.Many2one("res.users", default=lambda self: self.env.user)
    payload_preview = fields.Text(readonly=True, copy=False)

    @api.model_create_multi
    def create(self, vals_list):
        for vals in vals_list:
            if vals.get("name", "/") == "/":
                vals["name"] = self.env["ir.sequence"].next_by_code("eg.trade.eta.document") or "/"
        return super().create(vals_list)

    def action_check_ready(self):
        for document in self:
            missing = document._collect_missing_data()
            if missing:
                document.write({"state": "rejected", "last_error": "\n".join(missing)})
                return {
                    "type": "ir.actions.client",
                    "tag": "display_notification",
                    "params": {
                        "title": _("Missing Egypt tax data"),
                        "message": "\n".join(missing),
                        "type": "warning",
                        "sticky": True,
                    },
                }
            document.write({"state": "ready", "last_error": False, "payload_preview": document._build_payload_preview()})
        return True

    def action_mark_sent(self):
        now = fields.Datetime.now()
        for document in self:
            if document.state not in ("ready", "rejected", "retry"):
                raise UserError(_("Only ready, rejected, or retry documents can be sent."))
            document.write({
                "state": "sent",
                "retry_count": document.retry_count + 1,
                "last_send_date": now,
                "last_error": False,
            })
        return True

    def action_mark_accepted(self):
        for document in self:
            document.write({"state": "accepted", "last_error": False})
            if document.move_id:
                document.move_id.eg_trade_eta_state = "accepted"
                document.move_id.eg_trade_eta_uuid = document.eta_uuid
        return True

    def action_need_retry(self):
        self.write({"state": "retry"})
        return True

    def action_cancel(self):
        self.write({"state": "cancelled"})
        return True

    def action_open_source(self):
        self.ensure_one()
        if not self.source_model or not self.source_res_id:
            return False
        return {
            "type": "ir.actions.act_window",
            "name": _("Source Document"),
            "res_model": self.source_model,
            "res_id": self.source_res_id,
            "view_mode": "form",
            "target": "current",
        }

    def _collect_missing_data(self):
        self.ensure_one()
        missing = []
        partner = self.partner_id or self.move_id.partner_id or self.sale_order_id.partner_id
        if not partner:
            missing.append(_("Customer is missing."))
            return missing
        if self.document_type in ("invoice", "credit_note", "debit_note") and not partner.eg_trade_compliance_ready:
            missing.append(_("Customer tax profile is incomplete."))
        lines = self.move_id.invoice_line_ids if self.move_id else self.sale_order_id.order_line
        product_templates = lines.mapped("product_id.product_tmpl_id")
        products_without_codes = product_templates.filtered(lambda product: not product.eg_trade_ready)
        if products_without_codes:
            missing.append(_("Some products are missing Egypt tax codes or readiness data."))
        return missing

    def _build_payload_preview(self):
        self.ensure_one()
        partner = self.partner_id or self.move_id.partner_id or self.sale_order_id.partner_id
        return "\n".join(
            [
                "Document: %s" % self.name,
                "Type: %s" % self.document_type,
                "Customer: %s" % (partner.display_name if partner else ""),
                "Amount: %s" % self.amount_total,
                "State: ready for connector implementation",
            ]
        )
