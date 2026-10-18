# -*- coding: utf-8 -*-
"""
APK Download Validator — OCP Foundation
=======================================
Validates gated APK download submissions (BDD-style rules, Arabic-first).
"""
import re

from ..constants import AppConfig, ValidationRegex
from .base_validator import BaseValidator, ValidationResult


class ApkDownloadValidator(BaseValidator):
    """Validates APK download request form data."""

    MSG_APK = 'الرجاء اختيار التطبيق المطلوب تحميله.'
    MSG_NAME = 'الاسم يجب أن يتراوح بين 2 و 100 حرف.'
    MSG_EMAIL = 'يرجى إدخال بريد إلكتروني صحيح.'
    MSG_PHONE = 'رقم الهاتف يجب أن يتكون من 7 إلى 20 رقماً مع إمكانية البدء بـ +.'

    def validate(self, data: dict) -> ValidationResult:
        result = ValidationResult()
        self._validate_apk(data.get('apk_id'), result)
        self._validate_name(data.get('name', ''), result)
        self._validate_email(data.get('email', ''), result)
        self._validate_phone(data.get('phone', ''), result)
        return result

    def _validate_apk(self, apk_id, result):
        try:
            apk_id = int(apk_id)
        except (TypeError, ValueError):
            result.add_error('apk_id', self.MSG_APK)
            return
        if apk_id <= 0:
            result.add_error('apk_id', self.MSG_APK)

    def _validate_name(self, name, result):
        if not name or not (AppConfig.CONTACT_FORM_MIN_NAME <= len(name) <= AppConfig.CONTACT_FORM_MAX_NAME):
            result.add_error('name', self.MSG_NAME)
        elif not re.match(ValidationRegex.NAME, name):
            result.add_error('name', self.MSG_NAME)

    def _validate_email(self, email, result):
        if not email or not re.match(ValidationRegex.EMAIL, email):
            result.add_error('email', self.MSG_EMAIL)

    def _validate_phone(self, phone, result):
        phone = (phone or '').strip()
        if phone and not re.match(ValidationRegex.PHONE, phone):
            result.add_error('phone', self.MSG_PHONE)