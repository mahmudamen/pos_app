# -*- coding: utf-8 -*-
"""
Contact Service — CRM Integration Layer
========================================
Encapsulates lead creation from contact/demo requests.
Single Responsibility: CRM operations only.
"""
from typing import Optional, Any
from ..constants import AppConfig


class ContactService:
    """Single Responsibility: manage CRM lead creation"""

    def __init__(self, env):
        """Inject Odoo environment"""
        self._env = env
        self._crm_lead = env['crm.lead']
        self._crm_team = env['crm.team']
        self._crm_tag = env['crm.tag']

    def create_contact_lead(self, name: str, email: str, phone: str,
                            subject: str, message: str) -> bool:
        """
        Create a contact lead in CRM (from contact form).
        Args:
            name: contact name
            email: contact email
            phone: contact phone
            subject: inquiry subject
            message: inquiry message
        Returns:
            True if successful, False otherwise
        """
        try:
            team = self._get_or_create_team(AppConfig.CRM_TEAM_NAME)
            lead_data = {
                'name': f'[Contact] {subject or "General"} - {name}',
                'contact_name': name,
                'email_from': email,
                'phone': phone,
                'description': message,
                'type': 'lead',
            }
            if team:
                lead_data['team_id'] = team.id
            self._crm_lead.sudo().create(lead_data)
            return True
        except Exception:
            return False

    def create_demo_lead(self, name: str, whatsapp: str,
                         addon_interest: str) -> bool:
        """
        Create a demo request lead in CRM.
        Args:
            name: requester name
            whatsapp: whatsapp number
            addon_interest: interested addon or 'All'
        Returns:
            True if successful, False otherwise
        """
        try:
            team = self._get_or_create_team(AppConfig.CRM_TEAM_NAME)
            tag = self._get_or_create_tag(AppConfig.CRM_DEMO_TAG_NAME)

            lead_data = {
                'name': f'[Demo] {addon_interest or "POS"} - {name}',
                'contact_name': name,
                'phone': whatsapp,
                'description': (
                    f'Demo Request\n'
                    f'Name: {name}\n'
                    f'WhatsApp: {whatsapp}\n'
                    f'Interest: {addon_interest or "All POS"}\n'
                    f'URL: {AppConfig.DEMO_URL}\n'
                    f'Credentials: {AppConfig.DEMO_CREDENTIALS}'
                ),
                'type': 'lead',
            }
            if team:
                lead_data['team_id'] = team.id
            if tag:
                lead_data['tag_ids'] = [(4, tag.id)]

            self._crm_lead.sudo().create(lead_data)
            return True
        except Exception:
            return False

    def _get_or_create_team(self, team_name: str) -> Optional[Any]:
        """
        Get or create CRM team by name.
        Args:
            team_name: team name (e.g. 'Demo Requests')
        Returns:
            CRM team record or None
        """
        team = self._crm_team.sudo().search([('name', '=', team_name)], limit=1)
        if not team:
            team = self._crm_team.sudo().create({'name': team_name})
        return team

    def _get_or_create_tag(self, tag_name: str) -> Optional[Any]:
        """
        Get or create CRM tag by name.
        Args:
            tag_name: tag name (e.g. 'Demo Request')
        Returns:
            CRM tag record or None
        """
        tag = self._crm_tag.sudo().search([('name', '=', tag_name)], limit=1)
        if not tag:
            tag = self._crm_tag.sudo().create({'name': tag_name})
        return tag
