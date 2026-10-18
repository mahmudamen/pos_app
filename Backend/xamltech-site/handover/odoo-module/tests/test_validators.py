# -*- coding: utf-8 -*-
"""
Test Validators — BDD Form Validation Tests
=============================================
Tests for contact and demo form validators (BDD-style).
"""
from odoo.tests import TransactionCase, tagged
from ..validators.contact_validator import ContactValidator
from ..validators.demo_validator import DemoValidator


@tagged('xamltech_store', 'validators')
class TestContactValidator(TransactionCase):
    """BDD: Contact form validation rules"""

    def setUp(self):
        super().setUp()
        self.validator = ContactValidator()

    def test_valid_contact_passes_validation(self):
        result = self.validator.validate({
            'name': 'Mahmoud Amen',
            'email': 'mahmudamen@gmail.com',
            'phone': '+201234567890',
            'subject': 'Inquiry',
            'message': 'This is a valid message with sufficient length',
        })
        self.assertTrue(result.is_valid, f"Errors: {result.errors}")

    def test_empty_name_fails_validation(self):
        result = self.validator.validate({
            'name': '',
            'email': 'test@example.com',
            'phone': '+201234567890',
            'subject': 'Test',
            'message': 'This is a message',
        })
        self.assertFalse(result.is_valid)
        self.assertIn('name', result.errors)

    def test_name_too_short_fails_validation(self):
        result = self.validator.validate({
            'name': 'A',
            'email': 'test@example.com',
            'phone': '+201234567890',
            'subject': 'Test',
            'message': 'This is a message',
        })
        self.assertFalse(result.is_valid)
        self.assertIn('name', result.errors)

    def test_name_too_long_fails_validation(self):
        result = self.validator.validate({
            'name': 'A' * 101,
            'email': 'test@example.com',
            'phone': '+201234567890',
            'subject': 'Test',
            'message': 'This is a message',
        })
        self.assertFalse(result.is_valid)
        self.assertIn('name', result.errors)

    def test_invalid_email_fails_validation(self):
        result = self.validator.validate({
            'name': 'Test User',
            'email': 'not-an-email',
            'phone': '+201234567890',
            'subject': 'Test',
            'message': 'This is a message',
        })
        self.assertFalse(result.is_valid)
        self.assertIn('email', result.errors)

    def test_valid_email_passes_validation(self):
        result = self.validator.validate({
            'name': 'Test User',
            'email': 'test@example.co.uk',
            'phone': '+201234567890',
            'subject': 'Test',
            'message': 'This is a message',
        })
        self.assertTrue(result.is_valid)

    def test_short_message_fails_validation(self):
        result = self.validator.validate({
            'name': 'Test User',
            'email': 'test@example.com',
            'phone': '+201234567890',
            'subject': 'Test',
            'message': 'short',
        })
        self.assertFalse(result.is_valid)
        self.assertIn('message', result.errors)

    def test_invalid_phone_fails_validation(self):
        result = self.validator.validate({
            'name': 'Test User',
            'email': 'test@example.com',
            'phone': '123',
            'subject': 'Test',
            'message': 'This is a message',
        })
        self.assertFalse(result.is_valid)
        self.assertIn('phone', result.errors)

    def test_empty_phone_fails_validation(self):
        result = self.validator.validate({
            'name': 'Test User',
            'email': 'test@example.com',
            'phone': '',
            'subject': 'Test',
            'message': 'This is a message',
        })
        self.assertFalse(result.is_valid)
        self.assertIn('phone', result.errors)

    def test_missing_subject_fails_validation(self):
        result = self.validator.validate({
            'name': 'Test User',
            'email': 'test@example.com',
            'phone': '+201234567890',
            'subject': '',
            'message': 'This is a message',
        })
        self.assertFalse(result.is_valid)
        self.assertIn('subject', result.errors)

    def test_valid_phone_with_plus_prefix_passes(self):
        result = self.validator.validate({
            'name': 'Test User',
            'email': 'test@example.com',
            'phone': '+201234567890',
            'subject': 'Test',
            'message': 'This is a message',
        })
        self.assertTrue(result.is_valid)

    def test_valid_phone_without_plus_prefix_passes(self):
        result = self.validator.validate({
            'name': 'Test User',
            'email': 'test@example.com',
            'phone': '201234567890',
            'subject': 'Test',
            'message': 'This is a message',
        })
        self.assertTrue(result.is_valid)


@tagged('xamltech_store', 'validators')
class TestDemoValidator(TransactionCase):
    """BDD: Demo request form validation rules"""

    def setUp(self):
        super().setUp()
        self.validator = DemoValidator()

    def test_valid_demo_request_passes_validation(self):
        result = self.validator.validate({
            'name': 'User Name',
            'whatsapp': '+201234567890',
            'addon_interest': 'POS System',
        })
        self.assertTrue(result.is_valid, f"Errors: {result.errors}")

    def test_empty_name_fails_validation(self):
        result = self.validator.validate({
            'name': '',
            'whatsapp': '+201234567890',
        })
        self.assertFalse(result.is_valid)
        self.assertIn('name', result.errors)

    def test_name_too_short_fails_validation(self):
        result = self.validator.validate({
            'name': 'A',
            'whatsapp': '+201234567890',
        })
        self.assertFalse(result.is_valid)
        self.assertIn('name', result.errors)

    def test_invalid_whatsapp_fails_validation(self):
        result = self.validator.validate({
            'name': 'User Name',
            'whatsapp': '123',
        })
        self.assertFalse(result.is_valid)
        self.assertIn('whatsapp', result.errors)

    def test_empty_whatsapp_fails_validation(self):
        result = self.validator.validate({
            'name': 'User Name',
            'whatsapp': '',
        })
        self.assertFalse(result.is_valid)
        self.assertIn('whatsapp', result.errors)

    def test_valid_whatsapp_with_plus_passes(self):
        result = self.validator.validate({
            'name': 'User Name',
            'whatsapp': '+201234567890',
        })
        self.assertTrue(result.is_valid)

    def test_valid_whatsapp_without_plus_passes(self):
        result = self.validator.validate({
            'name': 'User Name',
            'whatsapp': '201234567890',
        })
        self.assertTrue(result.is_valid)

    def test_addon_interest_is_optional(self):
        result = self.validator.validate({
            'name': 'User Name',
            'whatsapp': '+201234567890',
            'addon_interest': '',
        })
        self.assertTrue(result.is_valid)
