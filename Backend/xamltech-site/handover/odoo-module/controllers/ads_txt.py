# -*- coding: utf-8 -*-
from odoo import http
from odoo.http import request
import os

_ADDON_PATH = os.path.dirname(os.path.dirname(__file__))


class XamlTechAdsTxt(http.Controller):

    @http.route('/app-ads.txt', type='http', auth='public', website=True, sitemap=False, cors='*')
    def serve_ads_txt(self, **kw):
        file_path = os.path.join(_ADDON_PATH, 'static', 'app-ads.txt')

        if os.path.exists(file_path):
            with open(file_path, 'rb') as f:
                return request.make_response(
                    f.read(),
                    headers=[
                        ('Content-Type', 'text/plain; charset=utf-8'),
                        ('Content-Disposition', 'inline; filename=app-ads.txt'),
                    ],
                )
        return request.not_found()
