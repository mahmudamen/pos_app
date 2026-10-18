from odoo import _, fields, models
from odoo.exceptions import UserError


class AccountMove(models.Model):
    _inherit = "account.move"

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
        tracking=True,
        copy=False,
    )
    eg_trade_eta_uuid = fields.Char(string="ETA UUID", copy=False, index=True)
    eg_trade_eta_message = fields.Text(string="ETA Message", copy=False)
    eg_trade_document_type = fields.Selection(
        [
            ("invoice", "Invoice"),
            ("credit_note", "Credit Note"),
            ("debit_note", "Debit Note"),
            ("receipt", "Receipt"),
        ],
        string="Egypt Document Type",
        default="invoice",
    )
    eg_trade_eta_document_id = fields.Many2one("eg.trade.eta.document", string="Tax Document", copy=False)

    def action_eg_trade_prepare_eta(self):
        for move in self:
            if move.move_type not in ("out_invoice", "out_refund"):
                raise UserError(_("Only customer invoices and refunds can be prepared for Egypt tax submission."))
            document = move.eg_trade_eta_document_id or self.env["eg.trade.eta.document"].create({
                "document_type": "credit_note" if move.move_type == "out_refund" else move.eg_trade_document_type,
                "source_model": "account.move",
                "source_res_id": move.id,
                "move_id": move.id,
                "partner_id": move.partner_id.id,
                "amount_total": move.amount_total,
                "company_id": move.company_id.id,
            })
            move.eg_trade_eta_document_id = document.id
            result = document.action_check_ready()
            move.write({"eg_trade_eta_state": document.state, "eg_trade_eta_message": document.last_error})
            if isinstance(result, dict):
                return result
        return True

    def action_eg_trade_open_eta_document(self):
        self.ensure_one()
        if not self.eg_trade_eta_document_id:
            self.action_eg_trade_prepare_eta()
        return {
            "type": "ir.actions.act_window",
            "name": _("Egypt Tax Document"),
            "res_model": "eg.trade.eta.document",
            "res_id": self.eg_trade_eta_document_id.id,
            "view_mode": "form",
            "target": "current",
        }
