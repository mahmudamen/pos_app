# -*- coding: utf-8 -*-
"""
Test Contact Service — TDD CRM Integration Tests
=================================================
Tests for contact and demo lead creation.
"""
from odoo.tests import TransactionCase, tagged
from ..services.contact_service import ContactService
from ..constants import AppConfig


@tagged('xamltech_store', 'contact_service')
class TestContactService(TransactionCase):
    """Test CRM lead creation service"""

    def test_create_contact_lead_returns_true_on_success(self):
        service = ContactService(self.env)
        ok = service.create_contact_lead(
            name='Test User',
            email='test@example.com',
            phone='+201234567890',
            subject='Test Subject',
            message='This is a test message for inquiry',
        )
        self.assertTrue(ok)

    def test_create_contact_lead_creates_crm_lead(self):
        service = ContactService(self.env)
        initial_count = self.env['crm.lead'].search_count([])
        service.create_contact_lead(
            name='Test User',
            email='test@example.com',
            phone='+201234567890',
            subject='Test Subject',
            message='This is a test message for inquiry',
        )
        final_count = self.env['crm.lead'].search_count([])
        self.assertEqual(final_count, initial_count + 1)

    def test_create_contact_lead_sets_contact_name(self):
        service = ContactService(self.env)
        service.create_contact_lead(
            name='Test Contact',
            email='test@example.com',
            phone='+201234567890',
            subject='Subject',
            message='This is a test message',
        )
        lead = self.env['crm.lead'].search([('contact_name', '=', 'Test Contact')], limit=1)
        self.assertEqual(lead.contact_name, 'Test Contact')

    def test_create_contact_lead_includes_subject_in_name(self):
        service = ContactService(self.env)
        service.create_contact_lead(
            name='User',
            email='user@example.com',
            phone='+201234567890',
            subject='Important',
            message='This is a test message',
        )
        lead = self.env['crm.lead'].search([], order='create_date desc', limit=1)
        self.assertIn('Important', lead.name)

    def test_create_demo_lead_returns_true_on_success(self):
        service = ContactService(self.env)
        ok = service.create_demo_lead(
            name='Demo User',
            whatsapp='+201234567890',
            addon_interest='POS System',
        )
        self.assertTrue(ok)

    def test_create_demo_lead_creates_crm_lead(self):
        service = ContactService(self.env)
        initial_count = self.env['crm.lead'].search_count([])
        service.create_demo_lead(
            name='Demo User',
            whatsapp='+201234567890',
            addon_interest='POS System',
        )
        final_count = self.env['crm.lead'].search_count([])
        self.assertEqual(final_count, initial_count + 1)

    def test_create_demo_lead_includes_demo_tag(self):
        service = ContactService(self.env)
        service.create_demo_lead(
            name='Demo User',
            whatsapp='+201234567890',
            addon_interest='All',
        )
        lead = self.env['crm.lead'].search([], order='create_date desc', limit=1)
        tag_names = lead.tag_ids.mapped('name')
        self.assertIn(AppConfig.CRM_DEMO_TAG_NAME, tag_names)

    def test_create_demo_lead_creates_tag_if_not_exists(self):
        service = ContactService(self.env)
        initial_tag_count = self.env['crm.tag'].search_count(
            [('name', '=', AppConfig.CRM_DEMO_TAG_NAME)]
        )
        service.create_demo_lead(
            name='Demo User',
            whatsapp='+201234567890',
            addon_interest='System',
        )
        final_tag_count = self.env['crm.tag'].search_count(
            [('name', '=', AppConfig.CRM_DEMO_TAG_NAME)]
        )
        self.assertGreaterEqual(final_tag_count, initial_tag_count)

    def test_create_demo_lead_includes_description_with_details(self):
        service = ContactService(self.env)
        service.create_demo_lead(
            name='User Name',
            whatsapp='+201234567890',
            addon_interest='POS',
        )
        lead = self.env['crm.lead'].search([], order='create_date desc', limit=1)
        desc = lead.description
        self.assertIn('Demo Request', desc)
        self.assertIn('User Name', desc)
        self.assertIn('+201234567890', desc)
        self.assertIn('POS', desc)

    def test_get_or_create_team_returns_team(self):
        service = ContactService(self.env)
        team = service._get_or_create_team('Test Team')
        self.assertIsNotNone(team)
        self.assertEqual(team.name, 'Test Team')

    def test_get_or_create_team_creates_if_not_exists(self):
        service = ContactService(self.env)
        team_name = 'Unique Test Team ' + str(self.env.uid)
        initial_count = self.env['crm.team'].search_count([('name', '=', team_name)])
        team = service._get_or_create_team(team_name)
        final_count = self.env['crm.team'].search_count([('name', '=', team_name)])
        self.assertEqual(initial_count, 0)
        self.assertEqual(final_count, 1)

    def test_get_or_create_team_returns_existing(self):
        service = ContactService(self.env)
        team_name = 'Existing Team ' + str(self.env.uid)
        first_team = service._get_or_create_team(team_name)
        second_team = service._get_or_create_team(team_name)
        self.assertEqual(first_team.id, second_team.id)
