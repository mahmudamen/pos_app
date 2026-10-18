from odoo import api, fields, models


class ResPartner(models.Model):
    _inherit = "res.partner"

    eg_trade_customer_type = fields.Selection(
        [
            ("company", "Company"),
            ("person", "Person"),
            ("government", "Government"),
        ],
        string="Egypt Customer Type",
        default="company",
    )
    eg_trade_tax_registration = fields.Char(string="Egypt Tax Registration", index=True)
    eg_trade_national_id = fields.Char(string="National ID")
    eg_trade_legal_address = fields.Text(string="Legal Address")
    eg_trade_branch = fields.Char(string="Branch")
    eg_trade_e_invoice_enabled = fields.Boolean(string="Can Issue E-Invoice", default=True)
    eg_trade_currency_id = fields.Many2one("res.currency", default=lambda self: self.env.company.currency_id)
    eg_trade_credit_limit = fields.Monetary(string="Simple Credit Limit", currency_field="eg_trade_currency_id")
    eg_trade_compliance_ready = fields.Boolean(
        string="Egypt Compliance Ready",
        compute="_compute_eg_trade_compliance_ready",
        store=True,
    )

    @api.depends(
        "eg_trade_customer_type",
        "eg_trade_tax_registration",
        "eg_trade_national_id",
        "eg_trade_legal_address",
        "street",
        "city",
        "country_id",
    )
    def _compute_eg_trade_compliance_ready(self):
        for partner in self:
            has_address = bool(partner.eg_trade_legal_address or partner.street or partner.city or partner.country_id)
            if partner.eg_trade_customer_type in ("company", "government"):
                partner.eg_trade_compliance_ready = bool(partner.eg_trade_tax_registration and has_address)
            else:
                partner.eg_trade_compliance_ready = bool(has_address)
