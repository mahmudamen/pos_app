# -*- coding: utf-8 -*-
"""
APK Download Service
====================
Handles gated APK download requests and expiring-token verification.

Thin controllers delegate here; pure token math lives in ``../tools/apk_token``
so it stays importable from models without circular imports.
"""
from urllib.parse import quote

from ..tools.apk_token import (
    APK_SECRET_PARAM,
    datetime_to_ts,
    sign_download_token,
    verify_download_token,
)
from ..validators.apk_validator import ApkDownloadValidator
from ..constants import AppConfig


class ApkDownloadService:

    def __init__(self, env):
        self.env = env

    def get_secret(self):
        """Return (and lazily create) the HMAC signing secret."""
        config = self.env['ir.config_parameter'].sudo()
        secret = config.get_param(APK_SECRET_PARAM)
        if not secret:
            import os
            secret = os.urandom(32).hex()
            config.set_param(APK_SECRET_PARAM, secret)
        return secret

    def get_available_apks(self):
        """Active APKs visible in the public download form."""
        return self.env['xamltech.apk'].sudo().search(
            [('active', '=', True)],
            order='sequence, id desc',
        )

    def create_request(self, apk_id, name, email, phone='', android_version='', reason=''):
        """Validate and create a pending download request (public auth)."""
        validator = ApkDownloadValidator()
        result = validator.validate({
            'apk_id': apk_id,
            'name': name,
            'email': email,
            'phone': phone,
        })
        if not result.is_valid:
            return {'success': False, 'errors': result.errors}

        try:
            apk_id = int(apk_id)
        except (TypeError, ValueError):
            return {
                'success': False,
                'errors': {'apk_id': validator.MSG_APK},
            }

        apk = self.env['xamltech.apk'].sudo().browse(apk_id)
        if not apk.exists() or not apk.active:
            return {
                'success': False,
                'errors': {'apk_id': validator.MSG_APK},
            }

        request = self.env['xamltech.app.request'].sudo().create({
            'apk_id': apk.id,
            'applicant_name': str(name).strip(),
            'email': str(email).strip(),
            'phone': str(phone or '').strip(),
            'android_version': str(android_version or '').strip(),
            'reason': str(reason or '').strip(),
        })
        try:
            team = self.env['crm.team'].sudo().search(
                [('name', '=', AppConfig.CRM_TEAM_NAME)], limit=1)
            if not team:
                team = self.env['crm.team'].sudo().create(
                    {'name': AppConfig.CRM_TEAM_NAME})
            tag = self.env['crm.tag'].sudo().search(
                [('name', '=', AppConfig.CRM_APK_TAG_NAME)], limit=1)
            if not tag:
                tag = self.env['crm.tag'].sudo().create(
                    {'name': AppConfig.CRM_APK_TAG_NAME})
            lead_data = {
                'name': 'APK Request: {}'.format(name),
                'team_id': team.id,
                'tag_ids': [(6, 0, [tag.id])],
                'contact_name': name,
                'email_from': email,
                'description': (
                    'APK download request\n'
                    'Name: {}\n'
                    'Email: {}\n'
                    'Phone: {}\n'
                    'Android Version: {}\n'
                    'Reason: {}\n'.format(
                        name, email, phone, android_version, reason)),
            }
            lead = self.env['crm.lead'].sudo().create(lead_data)
            request.write({'lead_id': lead.id})
        except Exception:
            _logger.exception('Failed to create CRM lead for APK request request')

        return {'success': True, 'request_id': request.id, 'request': request}

    def verify_download(self, request_id, expiry_ts, token):
        """
        Validate the route token AND the request eligibility.

        Returns the request record for streaming, or ``None`` on any failure
        (bad/tampered/expired token, non-approved request, altered expiry,
        missing binary).
        """
        try:
            request_id = int(request_id)
            expiry_ts = int(expiry_ts)
        except (TypeError, ValueError):
            return None

        if not verify_download_token(self.get_secret(), request_id, expiry_ts, token):
            return None

        request = self.env['xamltech.app.request'].sudo().browse(request_id)
        if not request.exists() or request.state != 'approved':
            return None

        expected_exp = datetime_to_ts(request.expiry)
        if expected_exp is None or expected_exp != expiry_ts:
            return None

        if not request.apk_id.datas:
            return None

        return request

    def stream_download(self, request):
        """Render the APK bytes plus safe download headers."""
        request.ensure_one()
        data = request.apk_id.datas or b''
        filename = request.apk_id.file_name or 'app.apk'
        disposition = (
            "attachment; filename=%s; filename*=UTF-8''%s"
            % (quote(filename), quote(filename))
        )
        headers = [
            ('Content-Type', 'application/vnd.android.package-archive'),
            ('Content-Disposition', disposition),
            ('Content-Length', str(len(data))),
            ('X-Content-Type-Options', 'nosniff'),
            ('Cache-Control', 'no-store'),
        ]
        return data, headers
