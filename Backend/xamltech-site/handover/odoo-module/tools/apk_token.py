# -*- coding: utf-8 -*-
"""
APK Download Token Helpers
==========================
Pure HMAC-SHA256 expiring-token helpers for gated APK downloads.

This module MUST remain free of Odoo imports so it can be unit-tested
without a database and safely reused by models, services and controllers
without introducing circular imports.
"""
import hashlib
import hmac
import time
from datetime import timezone

# ir.config_parameter key holding the HMAC signing secret (generated lazily).
APK_SECRET_PARAM = 'xamltech_apk_secret'

# How many days an approved download link stays valid.
APK_DOWNLOAD_EXPIRY_DAYS = 30


def datetime_to_ts(naive_utc_dt):
    """Convert an Odoo naive-UTC Datetime to an integer epoch timestamp."""
    if not naive_utc_dt:
        return None
    return int(naive_utc_dt.replace(tzinfo=timezone.utc).timestamp())


def sign_download_token(secret, request_id, expiry_ts):
    """Build the HMAC-SHA256 signature for ``request_id:expiry_ts``."""
    payload = '%s:%s' % (request_id, expiry_ts)
    return hmac.new(
        str(secret).encode('utf-8'),
        payload.encode('utf-8'),
        hashlib.sha256,
    ).hexdigest()


def verify_download_token(secret, request_id, expiry_ts, provided, now_ts=None):
    """
    Verify ``provided`` against the canonical signature for the given payload.

    Rejects non-integer ids/timestamps, expired tokens and tampered values.
    Comparison is constant-time via ``hmac.compare_digest``.
    """
    if not provided or not isinstance(provided, str):
        return False
    try:
        request_id = int(request_id)
        expiry_ts = int(expiry_ts)
        now_ts = int(now_ts if now_ts is not None else time.time())
    except (TypeError, ValueError):
        return False
    if now_ts > expiry_ts:
        return False
    expected = sign_download_token(secret, request_id, expiry_ts)
    return hmac.compare_digest(expected, provided)