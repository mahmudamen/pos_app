# -*- coding: utf-8 -*-
from odoo import models, fields, api


class XamlTechAddonCategory(models.Model):
    _name = 'xamltech.addon.category'
    _description = 'Addon Category'
    _order = 'sequence, name'

    name = fields.Char(required=True, translate=True)
    icon_emoji = fields.Char(string='Icon', default='📁')
    sequence = fields.Integer(default=10)
    addon_ids = fields.One2many(
        'xamltech.addon', 'category_id', string='Addons',
    )
    addon_count = fields.Integer(
        compute='_compute_addon_count', string='# Addons',
    )
    description = fields.Text(translate=True)
    active = fields.Boolean(default=True)

    @api.depends('addon_ids')
    def _compute_addon_count(self):
        for cat in self:
            cat.addon_count = len(
                cat.addon_ids.filtered(lambda a: a.state == 'published')
            )
