# -*- coding: utf-8 -*-
"""
APK Download Request Models
===========================
Gated APK distribution:

* ``xamltech.apk``         -> app registry + APK binary + public listing card
* ``xamltech.app.request`` -> visitor download request with approval workflow

Arabic-first labels. Record deletion is blocked by design (audit rule);
apps are archived via ``active`` and requests stay immutable.
"""
import os
from datetime import timedelta

from odoo import api, fields, models, _
from odoo.exceptions import UserError

from ..tools.apk_token import (
    APK_DOWNLOAD_EXPIRY_DAYS,
    APK_SECRET_PARAM,
    datetime_to_ts,
    sign_download_token,
)


class XamlTechApk(models.Model):
    _name = 'xamltech.apk'
    _description = 'تطبيق أندرويد'
    _inherit = ['mail.thread']
    _order = 'sequence, id desc'

    name = fields.Char(
        string='اسم التطبيق',
        required=True,
        tracking=True,
        translate=True,
    )
    version = fields.Char(
        string='الإصدار',
        required=True,
        default='1.0.0',
        tracking=True,
    )
    description = fields.Html(
        string='وصف التطبيق',
        translate=True,
    )
    file_name = fields.Char(
        string='اسم ملف APK',
        required=True,
        default='app-release.apk',
        help='اسم الملف الذي يظهر للمستخدم عند التحميل.',
    )
    datas = fields.Binary(
        string='ملف APK',
        required=True,
        attachment=True,
    )
    sequence = fields.Integer(string='الترتيب', default=10)
    active = fields.Boolean(string='مفعّل', default=True, tracking=True)
    request_ids = fields.One2many(
        'xamltech.app.request', 'apk_id', string='طلبات التحميل',
    )
    request_count = fields.Integer(
        compute='_compute_request_stats', string='عدد الطلبات',
    )
    approved_count = fields.Integer(
        compute='_compute_request_stats', string='الطلبات المعتمدة',
    )

    @api.depends('request_ids.state')
    def _compute_request_stats(self):
        for rec in self:
            rec.request_count = len(rec.request_ids)
            rec.approved_count = len(
                rec.request_ids.filtered(lambda r: r.state == 'approved')
            )

    def unlink(self):
        raise UserError(
            _('لا يمكن حذف التطبيقات. قم بإلغاء التفعيل (أرشفة) بدلاً من الحذف.')
        )


class XamlTechAppRequest(models.Model):
    _name = 'xamltech.app.request'
    lead_id = fields.Many2one(
        'crm.lead', string='CRM Lead', ondelete='set null')
    _description = 'طلب تحميل تطبيق'
    _inherit = ['mail.thread', 'mail.activity.mixin']
    _order = 'create_date desc'

    name = fields.Char(
        string='رقم الطلب',
        required=True,
        readonly=True,
        copy=False,
        index=True,
    )
    apk_id = fields.Many2one(
        'xamltech.apk',
        string='التطبيق',
        required=True,
        ondelete='restrict',
        tracking=True,
        index=True,
    )
    applicant_name = fields.Char(
        string='الاسم', required=True, tracking=True,
    )
    email = fields.Char(
        string='البريد الإلكتروني', required=True, tracking=True,
    )
    phone = fields.Char(
        string='رقم الهاتف', tracking=True,
    )
    android_version = fields.Char(
        string='إصدار نظام أندرويد', tracking=True,
    )
    reason = fields.Text(
        string='سبب التحميل', tracking=True,
    )
    state = fields.Selection([
        ('pending', 'قيد المراجعة'),
        ('approved', 'معتمد'),
        ('rejected', 'مرفوض'),
    ], string='الحالة', default='pending', required=True, tracking=True)
    expiry = fields.Datetime(
        string='الرابط صالح حتى',
        readonly=True,
        copy=False,
        tracking=True,
    )
    rejection_reason = fields.Text(
        string='سبب الرفض', tracking=True,
    )
    download_url = fields.Char(
        string='رابط التحميل',
        compute='_compute_download_url',
        help='رابط آمن منتهي الصلاحية لتحميل التطبيق.',
    )

    _name_unique = models.Constraint(
        'UNIQUE(name)',
        'رقم الطلب يجب أن يكون فريداً!',
    )

    @api.model_create_multi
    def create(self, vals_list):
        for vals in vals_list:
            if not vals.get('name'):
                vals['name'] = self.env['ir.sequence'].next_by_code(
                    'xamltech.app.request.seq'
                )
            if vals.get('email'):
                vals['email'] = str(vals['email']).strip().lower()
        return super().create(vals_list)

    @api.depends('state', 'expiry', 'apk_id.datas')
    def _compute_download_url(self):
        secret = self._get_download_secret()
        for rec in self:
            if rec.state == 'approved' and rec.expiry and rec.apk_id.datas:
                exp_ts = datetime_to_ts(rec.expiry)
                rec.download_url = '/apk/download/%s/%s/%s' % (
                    rec.id,
                    exp_ts,
                    sign_download_token(secret, rec.id, exp_ts),
                )
            else:
                rec.download_url = False

    @api.model
    def _get_download_secret(self):
        """Return (and lazily create) the HMAC signing secret."""
        config = self.env['ir.config_parameter'].sudo()
        secret = config.get_param(APK_SECRET_PARAM)
        if not secret:
            secret = os.urandom(32).hex()
            config.set_param(APK_SECRET_PARAM, secret)
        return secret

    def action_approve(self):
        for rec in self:
            if rec.state != 'pending':
                raise UserError(_('يمكن اعتماد الطلبات قيد المراجعة فقط.'))
            if not rec.apk_id.datas:
                raise UserError(_('لا يوجد ملف APK مرفق بالتطبيق.'))
            rec.with_context(tracking_disable=False).write({
                'state': 'approved',
                'expiry': fields.Datetime.now() + timedelta(
                    days=APK_DOWNLOAD_EXPIRY_DAYS
                ),
                'rejection_reason': False,
            })
            rec.message_post(body=_(
                'تم اعتماد الطلب %s. رابط التحميل صالح لمدة %s يوماً.',
                rec.name,
                APK_DOWNLOAD_EXPIRY_DAYS,
            ))

    def action_reject(self):
        for rec in self:
            if rec.state != 'pending':
                raise UserError(_('يمكن رفض الطلبات قيد المراجعة فقط.'))
            rec.with_context(tracking_disable=False).write({
                'state': 'rejected',
                'expiry': False,
            })
            rec.message_post(body=_('تم رفض الطلب %s.', rec.name))

    def unlink(self):
        raise UserError(
            _('لا يمكن حذف طلبات التحميل؛ يُحفظ السجل بشكل دائم.')
        )