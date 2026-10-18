from odoo import http
from odoo.http import request, content_disposition
import os


class XamlTechVerification(http.Controller):

    @http.route('/BingSiteAuth.xml', type='http', auth='public', website=True, sitemap=False)
    def serve_verification_file(self, **kw):
        addon_path = http.addons_manifest.get('xamltech_addons_store', {}).get('addons_path', '')
        file_path = os.path.join(addon_path, 'xamltech_addons_store', 'static', 'BingSiteAuth.xml')

        if os.path.exists(file_path):
            with open(file_path, 'rb') as f:
                return request.make_response(
                    f.read(),
                    headers=[
                        ('Content-Type', 'application/xml'),
                        ('Content-Disposition', 'inline; filename=BingSiteAuth.xml')
                    ]
                )
        return request.not_found()