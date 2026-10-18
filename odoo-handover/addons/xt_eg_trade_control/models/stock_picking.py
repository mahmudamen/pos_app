from odoo import api, fields, models


class StockPicking(models.Model):
    _inherit = "stock.picking"

    eg_trade_branch = fields.Char(string="Branch")
    eg_trade_simple_operation = fields.Selection(
        [
            ("receipt", "Receive"),
            ("delivery", "Deliver"),
            ("transfer", "Transfer"),
            ("adjustment", "Adjustment"),
            ("return", "Return"),
            ("damage", "Damage"),
        ],
        string="Simple Operation",
        compute="_compute_eg_trade_simple_operation",
        store=True,
    )
    eg_trade_operator_note = fields.Char(string="Warehouse Note")

    @api.depends("picking_type_id.code")
    def _compute_eg_trade_simple_operation(self):
        for picking in self:
            code = picking.picking_type_id.code
            if code == "incoming":
                picking.eg_trade_simple_operation = "receipt"
            elif code == "outgoing":
                picking.eg_trade_simple_operation = "delivery"
            elif code == "internal":
                picking.eg_trade_simple_operation = "transfer"
            else:
                picking.eg_trade_simple_operation = "adjustment"
