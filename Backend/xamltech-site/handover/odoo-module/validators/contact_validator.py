# -*- coding: utf-8 -*-
"""
Contact Validator — Contact Form Validation
==============================================
Validates contact form submissions with BDD-style rules.
Single Responsibility: contact form validation only.
"""
import re
from .base_validator import BaseValidator, ValidationResult
from ..constants import ErrorMessage, ValidationRegex, AppConfig


class ContactValidator(BaseValidator):
    """Validates contact form: name, email, phone, subject, message"""

    def validate(self, data: dict) -> ValidationResult:
        """
        Validate contact form data.
        Args:
            data: dict with keys: name, email, phone, subject, message
        Returns:
            ValidationResult with errors if invalid
        """
        result = ValidationResult()
        self._validate_name(data.get('name', ''), result)
        self._validate_email(data.get('email', ''), result)
        self._validate_phone(data.get('phone', ''), result)
        self._validate_subject(data.get('subject', ''), result)
        self._validate_message(data.get('message', ''), result)
        return result

    def _validate_name(self, name: str, result: ValidationResult):
        """BDD: name must be 2-100 chars, alphanumeric with spaces/hyphens/apostrophes"""
        if not name or len(name) < AppConfig.CONTACT_FORM_MIN_NAME or \
           len(name) > AppConfig.CONTACT_FORM_MAX_NAME:
            result.add_error('name', ErrorMessage.INVALID_NAME)
        elif not re.match(ValidationRegex.NAME, name):
            result.add_error('name', ErrorMessage.INVALID_NAME)

    def _validate_email(self, email: str, result: ValidationResult):
        """BDD: email must match standard email regex"""
        if not email or not re.match(ValidationRegex.EMAIL, email):
            result.add_error('email', ErrorMessage.INVALID_EMAIL)

    def _validate_phone(self, phone: str, result: ValidationResult):
        """BDD: phone must be 7-20 digits with optional + prefix"""
        if not phone or not re.match(ValidationRegex.PHONE, phone):
            result.add_error('phone', ErrorMessage.INVALID_PHONE)

    def _validate_subject(self, subject: str, result: ValidationResult):
        """BDD: subject is required, non-empty"""
        if not subject or not subject.strip():
            result.add_error('subject', ErrorMessage.INVALID_SUBJECT)

    def _validate_message(self, message: str, result: ValidationResult):
        """BDD: message must be at least 10 chars"""
        if not message or len(message) < AppConfig.CONTACT_FORM_MIN_MESSAGE:
            result.add_error('message', ErrorMessage.INVALID_MESSAGE)
