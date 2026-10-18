# -*- coding: utf-8 -*-
from . import controllers
from . import models


def _post_init_hook(env):
    import base64, os, logging
    _logger = logging.getLogger(__name__)
    Menu = env['website.menu']
    website = env['website'].browse(1)
    if not website.exists():
        return

    d = os.path.dirname(os.path.abspath(__file__))

    # Website
    website.write({'name': 'XamlTech', 'domain': 'https://xamltech.com'})
    fp = os.path.join(d, 'static', 'img', 'favicon.png')
    if os.path.exists(fp):
        try:
            with open(fp, 'rb') as f:
                website.write({'favicon': f.read()})
        except Exception as e:
            _logger.warning("XamlTech: Could not set favicon: %s", e)

    # Company
    c = env['res.company'].browse(1)
    if c.exists():
        v = {'name': 'XamlTech', 'email': 'mahmudamen@gmail.com',
             'phone': '+201007982468', 'website': 'https://xamltech.com',
             'street': 'Sahary', 'city': 'Aswan',
             'country_id': env.ref('base.eg').id}
        lp = os.path.join(d, 'static', 'img', 'logo.png')
        if os.path.exists(lp):
            try:
                with open(lp, 'rb') as f:
                    v['logo'] = f.read()
            except Exception as e:
                _logger.warning("XamlTech: Could not set logo: %s", e)
        c.write(v)

    # Menus - find root, delete safely, create new
    root = Menu.search([('website_id', '=', website.id), ('parent_id', '=', False)], limit=1)
    if not root:
        try:
            root = env.ref('website.main_menu')
        except Exception:
            return

    for m in Menu.search([('parent_id', '=', root.id)]):
        try:
            m.unlink()
        except Exception:
            try:
                m.write({'sequence': 9999})
            except Exception:
                pass

    # Create menu structure
    Menu.create({'name': 'Home', 'url': '/', 'parent_id': root.id, 'website_id': website.id, 'sequence': 10})

    sol = Menu.create({'name': 'Solutions', 'url': '#', 'parent_id': root.id, 'website_id': website.id, 'sequence': 20})
    Menu.create({'name': 'ERP & Odoo', 'url': '/solutions/erp', 'parent_id': sol.id, 'website_id': website.id, 'sequence': 1})
    Menu.create({'name': 'IT Infrastructure', 'url': '/solutions/it-infrastructure', 'parent_id': sol.id, 'website_id': website.id, 'sequence': 2})
    Menu.create({'name': 'Web Development', 'url': '/solutions/web', 'parent_id': sol.id, 'website_id': website.id, 'sequence': 3})
    Menu.create({'name': 'Odoo POS', 'url': '/odoo-pos-egypt', 'parent_id': sol.id, 'website_id': website.id, 'sequence': 4})
    Menu.create({'name': 'Agriculture ERP', 'url': '/erp-agriculture-egypt', 'parent_id': sol.id, 'website_id': website.id, 'sequence': 5})

    Menu.create({'name': 'Addons Store', 'url': '/addons-store', 'parent_id': root.id, 'website_id': website.id, 'sequence': 30})
    Menu.create({'name': 'Download Apps', 'url': '/apk-download', 'parent_id': root.id, 'website_id': website.id, 'sequence': 32})
    Menu.create({'name': 'Portfolio', 'url': '/portfolio', 'parent_id': root.id, 'website_id': website.id, 'sequence': 40})
    Menu.create({'name': 'Knowledge', 'url': '/knowledge', 'parent_id': root.id, 'website_id': website.id, 'sequence': 45})
    Menu.create({'name': 'Request Demo', 'url': '/request-demo', 'parent_id': root.id, 'website_id': website.id, 'sequence': 50})
    Menu.create({'name': 'Contact Us', 'url': '/contact-us', 'parent_id': root.id, 'website_id': website.id, 'sequence': 60})

    if not env['crm.team'].search([('name', '=', 'Demo Requests')], limit=1):
        env['crm.team'].create({'name': 'Demo Requests', 'sequence': 20})

    # Install Arabic language and set as default, keep English as secondary
    try:
        # Try to find Arabic - could be ar_001, ar_EG, or ar
        ar_lang = env['res.lang'].search([('code', 'in', ['ar_001', 'ar_EG', 'ar'])], limit=1)
        if not ar_lang:
            # Try to activate it
            try:
                wiz = env['base.language.install'].create({'overwrite': False})
                # Try different lang refs
                for ref in ['base.lang_ar', 'base.lang_ar_001']:
                    try:
                        lang_rec = env.ref(ref)
                        wiz.write({'lang_ids': [(4, lang_rec.id)]})
                        wiz.lang_install()
                        break
                    except Exception:
                        continue
            except Exception:
                pass
            ar_lang = env['res.lang'].search([('code', 'in', ['ar_001', 'ar_EG', 'ar'])], limit=1)

        if ar_lang:
            # Set Arabic as default website language
            website.write({'default_lang_id': ar_lang.id})
            _logger.info("XamlTech: Arabic (%s) set as default language", ar_lang.code)

        # Ensure English is also active
        en_lang = env['res.lang'].search([('code', '=', 'en_US')], limit=1)
        if en_lang and not en_lang.active:
            en_lang.write({'active': True})

        # Add both languages to the website
        if ar_lang and en_lang:
            website.write({'language_ids': [(6, 0, [ar_lang.id, en_lang.id])]})
            _logger.info("XamlTech: Website languages set to Arabic + English")
    except Exception as e:
        _logger.warning("XamlTech: Could not set Arabic language: %s", e)
