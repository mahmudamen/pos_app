# -*- coding: utf-8 -*-
"""
APK Download Tests — TDD Unit Tests
====================================
Covers the pure HMAC token helpers, the Arabic form validator, and the
gated download flow (create -> approve -> verify -> stream).

All user-facing messages asserted below are Arabic by design.
"""
import base64
from datetime import datetime, timezone

from odoo import exceptions
from odoo.tests import TransactionCase, tagged

from ..services.apk_download_service import ApkDownloadService
from ..tools.apk_token import (
    APK_DOWNLOAD_EXPIRY_DAYS,
    APK_SECRET_PARAM,
    datetime_to_ts,
    sign_download_token,
    verify_download_token,
)
from ..validators.apk_validator import ApkDownloadValidator

_DUMMY_APK_BYTES = b'PK\x03\x04 dummy apk payload'
_EMAIL = 'ahmed@xamltech.com'


@tagged('xamltech_store', 'apk_download')
class TestApkTokenUtils(TransactionCase):
    """Pure HMAC-SHA256 expiring-token helpers"""

    SECRET = 'test-secret-value'

    def _token(self, request_id=1, expiry_ts=None, secret=None):
        return sign_download_token(
            secret or self.SECRET,
            request_id,
            expiry_ts if expiry_ts is not None else 9999999999,
        )

    def test_sign_is_hex(self):
        token = self._token()
        self.assertIsInstance(token, str)
        self.assertEqual(len(token), 64)

    def test_verify_valid_token(self):
        expiry = 9999999999
        token = self._token(expiry_ts=expiry)
        self.assertTrue(verify_download_token(self.SECRET, '1', expiry, token, now_ts=expiry - 10))

    def test_verify_rejects_expired(self):
        expiry = 1000000
        token = self._token(expiry_ts=expiry)
        self.assertFalse(verify_download_token(self.SECRET, '1', expiry, token, now_ts=expiry + 1))

    def test_verify_rejects_tampered(self):
        expiry = 9999999999
        token = self._token(expiry_ts=expiry)
        self.assertFalse(verify_download_token(self.SECRET, '1', expiry, token[:-1] + ('0' if token[-1] != '0' else '1'), now_ts=0))

    def test_verify_rejects_wrong_request_id(self):
        expiry = 9999999999
        token = self._token(request_id=1, expiry_ts=expiry)
        self.assertFalse(verify_download_token(self.SECRET, '2', expiry, token, now_ts=0))

    def test_verify_rejects_wrong_secret(self):
        expiry = 9999999999
        token = self._token(expiry_ts=expiry)
        self.assertFalse(verify_download_token('other-secret', '1', expiry, token, now_ts=0))

    def test_verify_rejects_non_string_token(self):
        self.assertFalse(verify_download_token(self.SECRET, '1', 9999999999, 12345, now_ts=0))

    def test_verify_rejects_altered_expiry(self):
        expiry = 9999999999
        token = self._token(expiry_ts=expiry)
        # The signature itself is bound to the exact expiry value.
        self.assertFalse(verify_download_token(self.SECRET, '1', expiry + 1, token, now_ts=0))

    def test_datetime_to_ts_returns_none_for_empty(self):
        self.assertIsNone(datetime_to_ts(None))
        self.assertIsNone(datetime_to_ts(False))

    def test_datetime_to_ts_handles_naive_utc(self):
        dt = datetime(2026, 1, 1, tzinfo=timezone.utc).replace(tzinfo=None)
        ts = datetime_to_ts(dt)
        self.assertEqual(dt.replace(tzinfo=timezone.utc).timestamp(), ts)


@tagged('xamltech_store', 'apk_download')
class TestApkValidator(TransactionCase):
    """Arabic-first download request form validator"""

    def setUp(self):
        super().setUp()
        self.validator = ApkDownloadValidator()

    def _validate(self, **kw):
        data = {
            'apk_id': 1,
            'name': 'أحمد محمد',
            'email': _EMAIL,
            'phone': '',
        }
        data.update(kw)
        return self.validator.validate(data)

    def test_valid_data_passes(self):
        self.assertTrue(self._validate().is_valid)

    def test_valid_data_with_phone_passes(self):
        self.assertTrue(self._validate(phone='+201007982468').is_valid)

    def test_missing_email_fails(self):
        result = self._validate(email='')
        self.assertFalse(result.is_valid)
        self.assertIn('email', result.errors)

    def test_invalid_email_fails(self):
        result = self._validate(email='not-an-email')
        self.assertFalse(result.is_valid)
        self.assertIn('email', result.errors)

    def test_missing_apk_fails(self):
        result = self._validate(apk_id='')
        self.assertFalse(result.is_valid)
        self.assertIn('apk_id', result.errors)

    def test_non_positive_apk_fails(self):
        result = self._validate(apk_id=0)
        self.assertFalse(result.is_valid)
        self.assertIn('apk_id', result.errors)

    def test_short_name_fails(self):
        result = self._validate(name='م')
        self.assertFalse(result.is_valid)
        self.assertIn('name', result.errors)

    def test_bad_phone_fails_when_provided(self):
        result = self._validate(phone='123')
        self.assertFalse(result.is_valid)
        self.assertIn('phone', result.errors)


@tagged('xamltech_store', 'apk_download')
class TestApkDownloadFlow(TransactionCase):
    """Create -> approve -> verify -> stream gated download flow"""

    @classmethod
    def setUpClass(cls):
        super().setUpClass()
        cls.env['ir.config_parameter'].set_param(
            APK_SECRET_PARAM, 'flow-test-secret'
        )
        cls.apk = cls.env['xamltech.apk'].create({
            'name': 'تطبيق للحسابات',
            'version': '3.2.1',
            'description': 'تطبيق تجريبي للاختبارات',
            'file_name': 'app-release.apk',
            'datas': base64.b64encode(_DUMMY_APK_BYTES).decode('utf-8'),
        })
        cls.apk_archived = cls.env['xamltech.apk'].create({
            'name': 'تطبيق مؤرشف',
            'version': '1.0.0',
            'datas': base64.b64encode(_DUMMY_APK_BYTES).decode('utf-8'),
            'active': False,
        })

    def setUp(self):
        super().setUp()
        self.service = ApkDownloadService(self.env)

    def _create_request(self, **kw):
        data = {
            'apk_id': self.apk.id,
            'name': 'أحمد محمد',
            'email': _EMAIL,
        }
        data.update(kw)
        return self.service.create_request(**data)

    def test_create_request_returns_success_and_pending(self):
        result = self._create_request()
        self.assertTrue(result['success'])
        request = result['request']
        self.assertEqual(request.state, 'pending')
        self.assertTrue(request.name.startswith('APK/'))

    def test_create_request_has_no_download_url_while_pending(self):
        request = self._create_request()['request']
        self.assertFalse(request.download_url)

    def test_create_request_email_is_lowercased(self):
        request = self._create_request(email='Ahmed@XamlTech.COM')['request']
        self.assertEqual(request.email, _EMAIL)

    def test_create_request_rejects_invalid_email(self):
        result = self._create_request(email='broken')
        self.assertFalse(result['success'])
        self.assertIn('email', result['errors'])

    def test_create_request_rejects_unknown_apk(self):
        result = self._create_request(apk_id=999999)
        self.assertFalse(result['success'])
        self.assertIn('apk_id', result['errors'])

    def test_create_request_rejects_inactive_apk(self):
        result = self._create_request(apk_id=self.apk_archived.id)
        self.assertFalse(result['success'])
        self.assertIn('apk_id', result['errors'])

    def test_get_available_apks_returns_active_only(self):
        apks = self.service.get_available_apks()
        self.assertIn(self.apk.id, apks.ids)
        self.assertNotIn(self.apk_archived.id, apks.ids)

    def test_approve_sets_expiry_and_url(self):
        request = self._create_request()['request']
        request.action_approve()
        self.assertEqual(request.state, 'approved')
        self.assertTrue(request.expiry)
        self.assertTrue(request.download_url.startswith('/apk/download/'))

    def test_approve_requires_pending(self):
        request = self._create_request()['request']
        request.action_approve()
        with self.assertRaises(exceptions.UserError):
            request.action_approve()

    def test_reject_clears_expiry_and_url(self):
        request = self._create_request()['request']
        request.action_approve()
        request.action_reject()
        self.assertEqual(request.state, 'rejected')
        self.assertFalse(request.expiry)
        self.assertFalse(request.download_url)

    def test_verify_download_roundtrip(self):
        request = self._create_request()['request']
        request.action_approve()
        exp_ts = datetime_to_ts(request.expiry)
        token = sign_download_token(
            self.service.get_secret(), request.id, exp_ts
        )
        verified = self.service.verify_download(request.id, exp_ts, token)
        self.assertEqual(verified.id, request.id)

    def test_verify_download_rejects_wrong_token(self):
        request = self._create_request()['request']
        request.action_approve()
        exp_ts = datetime_to_ts(request.expiry)
        bad_token = sign_download_token(
            'wrong-secret', request.id, exp_ts
        )
        self.assertIsNone(self.service.verify_download(request.id, exp_ts, bad_token))

    def test_verify_download_rejects_expired_expiry(self):
        request = self._create_request()['request']
        request.action_approve()
        request.write({'expiry': datetime(2020, 1, 1)})
        request.refresh()
        exp_ts = datetime_to_ts(request.expiry)
        token = sign_download_token(
            self.service.get_secret(), request.id, exp_ts
        )
        self.assertIsNone(self.service.verify_download(request.id, exp_ts, token))

    def test_verify_download_rejects_pending(self):
        request = self._create_request()['request']
        exp_ts = 9999999999
        token = sign_download_token(
            self.service.get_secret(), request.id, exp_ts
        )
        self.assertIsNone(self.service.verify_download(request.id, exp_ts, token))

    def test_verify_download_rejects_non_int(self):
        self.assertIsNone(self.service.verify_download('abc', 123, 'token'))

    def test_stream_download_returns_bytes_and_headers(self):
        request = self._create_request()['request']
        request.action_approve()
        data, headers = self.service.stream_download(request)
        self.assertEqual(data, _DUMMY_APK_BYTES)
        header_dict = dict(headers)
        self.assertEqual(
            header_dict['Content-Type'],
            'application/vnd.android.package-archive',
        )
        self.assertEqual(header_dict['Content-Length'], str(len(_DUMMY_APK_BYTES)))

    def test_apk_unlink_is_blocked(self):
        with self.assertRaises(exceptions.UserError):
            self.apk.unlink()

    def test_request_unlink_is_blocked(self):
        request = self._create_request()['request']
        with self.assertRaises(exceptions.UserError):
            request.unlink()

    def test_approve_requires_attached_apk_file(self):
        bare = self.env['xamltech.apk'].create({
            'name': 'تطبيق بدون ملف',
            'version': '1.0.0',
        })
        request = self._create_request(apk_id=bare.id)['request']
        with self.assertRaises(exceptions.UserError):
            request.action_approve()