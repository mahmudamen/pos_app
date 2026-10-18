# -*- coding: utf-8 -*-
"""
Demo Validator — Demo Request Validation
=========================================
Validates demo request form submissions.
Single Responsibility: demo form validation only.
"""
import re
from .base_validator import BaseValidator, ValidationResult
from ..constants import ErrorMessage, ValidationRegex, AppConfig


class DemoValidator(BaseValidator):
    """Validates demo request form: name, whatsapp, addon_interest"""

    def validate(self, data: dict) -> ValidationResult:
        """
        Validate demo request form data.
        Args:
            data: dict with keys: name, whatsapp, addon_interest (optional)
        Returns:
            ValidationResult with errors if invalid
        """
        result = ValidationResult()
        self._validate_name(data.get('name', ''), result)
        self._validate_whatsapp(data.get('whatsapp', ''), result)
        return result

    def _validate_name(self, name: str, result: ValidationResult):
        """BDD: name must be 2-100 chars"""
        if not name or len(name) < AppConfig.DEMO_FORM_MIN_NAME or \
           len(name) > AppConfig.DEMO_FORM_MAX_NAME:
            result.add_error('name', ErrorMessage.INVALID_NAME)

    def _validate_whatsapp(self, whatsapp: str, result: ValidationResult):
        """BDD: whatsapp must be 7-20 digits with optional + prefix"""
        if not whatsapp or not re.match(ValidationRegex.PHONE, whatsapp):
            result.add_error('whatsapp', ErrorMessage.INVALID_WHATSAPP)
