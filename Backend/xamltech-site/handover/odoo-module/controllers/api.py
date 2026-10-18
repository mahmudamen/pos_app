# -*- coding: utf-8 -*-
"""
API Controller — Service Integration
======================================
All logic delegated to services and validators.
Single Responsibility: request/response handling only.
"""
import json
from odoo import http
from odoo.http import request
from ..services.addon_service import AddonService
from ..services.search_service import SearchService
from ..services.contact_service import ContactService
from ..validators.contact_validator import ContactValidator
from ..validators.demo_validator import DemoValidator


class XamltechApiController(http.Controller):
    """API routes — thin handlers delegating to services"""

    @http.route('/addons-store', type='http', auth='public', website=True, sitemap=True)
    def store_page(self, **kw):
        addon_service = AddonService(request.env)
        addons = addon_service.get_published()
        cats = request.env['xamltech.addon.category'].sudo().search(
            [('active', '=', True)], order='sequence'
        )
        return request.render('xamltech_addons_store.store_page', {
            'categories': cats,
            'addons': addons,
            'addons_data_json': json.dumps([addon_service.serialize(a) for a in addons]),
            'categories_data_json': json.dumps([
                {'id': c.id, 'name': c.name, 'icon': c.icon_emoji, 'count': c.addon_count}
                for c in cats
            ]),
            'stats': addon_service.get_store_stats(addons),
        })

    @http.route('/addons-store/<string:slug>', type='http', auth='public', website=True, sitemap=True)
    def addon_detail(self, slug, **kw):
        addon_service = AddonService(request.env)
        addon = addon_service.get_by_slug(slug)
        if not addon:
            return request.not_found()
        related = addon_service.get_related(addon)
        return request.render('xamltech_addons_store.addon_detail_page', {
            'addon': addon,
            'addon_data_json': json.dumps(addon_service.serialize(addon)),
            'related_addons': related,
        })

    @http.route('/addons-store/api/addons', type='jsonrpc', auth='public', methods=['POST'])
    def api_addons(self, category=None, search=None, sort='popular', page=1, limit=12):
        addon_service = AddonService(request.env)
        domain = SearchService.build_domain(category, search)
        order = SearchService.resolve_order(sort)

        addons = addon_service.get_published(
            domain=domain, order=order, limit=limit, offset=(page - 1) * limit
        )
        total = addon_service.count(domain)

        return {
            'addons': [addon_service.serialize(a) for a in addons],
            'total': total,
        }

    @http.route('/addons-store/api/track-click', type='jsonrpc', auth='public', methods=['POST'])
    def api_track(self, addon_id=None):
        addon_service = AddonService(request.env)
        if addon_id and addon_service.increment_download(addon_id):
            return {'success': True}
        return {'success': False}

    @http.route('/contact-us/send', type='jsonrpc', auth='public', methods=['POST'], csrf=False)
    def contact_send(self, name='', email='', phone='', subject='', message=''):
        validator = ContactValidator()
        result = validator.validate({
            'name': name,
            'email': email,
            'phone': phone,
            'subject': subject,
            'message': message,
        })

        if not result.is_valid:
            return {'success': False, 'errors': result.errors}

        contact_service = ContactService(request.env)
        ok = contact_service.create_contact_lead(name, email, phone, subject, message)
        return {'success': ok}

    @http.route('/request-demo/submit', type='jsonrpc', auth='public', methods=['POST'], csrf=False)
    def demo_submit(self, name='', whatsapp='', addon_interest=''):
        validator = DemoValidator()
        result = validator.validate({'name': name, 'whatsapp': whatsapp})

        if not result.is_valid:
            return {'success': False, 'errors': result.errors}

        contact_service = ContactService(request.env)
        ok = contact_service.create_demo_lead(name, whatsapp, addon_interest)
        return {'success': ok}
