# -*- coding: utf-8 -*-
"""
APK Download Controller
========================
* GET  /apk-download                -> public Arabic request page
* POST /apk-download/request        -> validate + create pending request
* GET  /apk/download/<rid>/<exp>/<token> -> gated streaming download

Arabic-first UI. Success/failure feedback via query params to avoid
re-POST on refresh, and the gated route always redirects on failure.
"""
from odoo import http
from odoo.http import request

from ..services.apk_download_service import ApkDownloadService


class ApkDownloadController(http.Controller):

    def _page_values(self, svc, **kw):
        error = kw.get('error') or ''
        if error == '1':
            error = 'رابط التحميل غير صالح أو منتهي الصلاحية.'
        return {
            'apks': svc.get_available_apks(),
            'sent': kw.get('sent') == '1',
            'error_msg': error,
            'form': kw,
        }

    @http.route('/apk-download', type='http', auth='public', website=True, sitemap=True)
    def apk_download_page(self, **kw):
        return request.render(
            'xamltech_addons_store.apk_download_page',
            self._page_values(ApkDownloadService(request.env), **kw),
        )

    @http.route('/apk-download/request', type='http', auth='public', website=True,
                csrf=True, methods=['POST'])
    def apk_request_submit(self, **post):
        svc = ApkDownloadService(request.env)
        result = svc.create_request(
            apk_id=post.get('apk_id'),
            name=post.get('name'),
            email=post.get('email'),
            phone=post.get('phone'),
            android_version=post.get('android_version'),
            reason=post.get('reason'),
        )
        if result['success']:
            return request.redirect('/apk-download?sent=1')
        post['error'] = self._first_error(result['errors'])
        return request.render(
            'xamltech_addons_store.apk_download_page',
            self._page_values(svc, **post),
        )

    @staticmethod
    def _first_error(errors):
        for field in ('apk_id', 'name', 'email', 'phone'):
            if field in errors:
                return errors[field]
        return next(iter(errors.values()), '')

    @http.route('/apk/download/<int:rid>/<int:exp>/<token>', type='http',
                auth='public', website=True, sitemap=False)
    def apk_download(self, rid, exp, token):
        svc = ApkDownloadService(request.env)
        req = svc.verify_download(rid, exp, token)
        if not req:
            return request.redirect('/apk-download?error=1')
        data, headers = svc.stream_download(req)
        return request.make_response(data, headers=headers)