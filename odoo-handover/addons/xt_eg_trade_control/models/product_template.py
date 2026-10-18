from odoo import api, fields, models


class ProductTemplate(models.Model):
    _inherit = "product.template"

    eg_trade_internal_code = fields.Char(string="Internal Trade Code", index=True)
    eg_trade_tax_code = fields.Char(string="Egypt Tax Code", index=True)
    eg_trade_tax_code_type = fields.Selection(
        [("egs", "EGS"), ("gs1", "GS1")],
        string="Tax Code Type",
        default="egs",
    )
    eg_trade_tax_category = fields.Selection(
        [
            ("standard", "Standard VAT"),
            ("exempt", "Exempt"),
            ("zero", "Zero Rated"),
            ("schedule", "Schedule Tax"),
            ("service", "Service"),
        ],
        string="Tax Category",
        default="standard",
    )
    eg_trade_default_vat_rate = fields.Float(string="Default VAT %", default=14.0)
    eg_trade_reorder_min = fields.Float(string="Simple Reorder Point")
    eg_trade_need_serial_or_lot = fields.Boolean(string="Needs Serial/Lot Tracking")
    eg_trade_ready = fields.Boolean(string="Egypt Trade Ready", compute="_compute_eg_trade_ready", store=True)

    @api.depends("eg_trade_tax_code", "uom_id", "list_price", "standard_price")
    def _compute_eg_trade_ready(self):
        for product in self:
            product.eg_trade_ready = bool(product.eg_trade_tax_code and product.uom_id and product.list_price >= 0)
