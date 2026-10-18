# -*- coding: utf-8 -*-
{
    'name': 'XamlTech Theme',
    'version': '19.0.9.0.0',
    'category': 'Website',
    'summary': 'Dark/Light RTL-ready professional website theme — Arabic-first, POS & ERP focused',
    'description': """
XamlTech Theme for Odoo 19 Website
====================================
A professional dark/light toggle website theme with full RTL (Arabic) support.
Built for technology companies, ERP consultancies, and digital agencies.

Features:
- Dark mode by default with one-click light mode toggle
- Full RTL layout for Arabic + bilingual (AR/EN) typography
- CSS custom properties (design tokens) — zero hardcoded colours
- Fluid typography with Cairo font (Arabic + Latin)
- Scroll-reveal animations and card micro-interactions
- PWA ready with Service Worker
- Hero, Stats, Cards, Steps, Timeline, Footer, WhatsApp float components
- Addons Store with live search, filter, sort
- Contact form with WhatsApp deep-link integration
- SEO structured data (JSON-LD Organization, Service, FAQPage)
    """,
    'author': 'Mahmoud Amen - XamlTech',
    'website': 'https://xamltech.com',
    'license': 'OPL-1',
    'price': 49.00,
    'currency': 'USD',
    'support': 'mahmudamen@gmail.com',
    'images': [
        'static/description/banner.png',
        'static/description/ss_dark.png',
        'static/description/ss_light.png',
        'static/description/ss_rtl.png',
        'static/description/ss_store.png',
        'static/description/ss_mobile.png',
        'static/description/theme_preview.gif',
    ],
    'depends': ['website', 'web', 'website_sale', 'website_blog', 'crm'],
    'data': [
        'security/ir.model.access.csv',
        'data/addon_category_data.xml',
        'data/addon_demo_data.xml',
        'data/blog_data.xml',
        'views/shared_blocks.xml',
        'views/homepage_templates.xml',
        'views/store_templates.xml',
        'views/erp_templates.xml',
        'views/infra_templates.xml',
        'views/web_dev_templates.xml',
        'views/portfolio_templates.xml',
        'views/contact_templates.xml',
        'views/demo_templates.xml',
        'views/seo_head.xml',
        'views/pwa_templates.xml',
        'views/seo_schemas.xml',
        'views/knowledge_templates.xml',
        'views/ai_apps_templates.xml',
        'views/blog_ai_articles.xml',
        'views/blog_seo_schemas.xml',
         'views/blog_hub.xml',
         'views/pos_page_templates.xml',
         'views/agri_page_templates.xml',
         'views/testimonials_templates.xml',
         'views/pricing_templates.xml',
         'views/aiseo_schemas.xml',
         'views/addon_views.xml',
         'views/theme_preview.xml',
         'views/networktool_templates.xml',
        'views/pos_go_templates.xml',
         'views/privacy_templates.xml',
         'views/website_pages.xml',
         'data/apk_sequence.xml',
         'views/apk_views.xml',
         'views/apk_templates.xml',
    ],
    'assets': {
        'web.assets_frontend': [
            'xamltech_addons_store/static/src/scss/xamltech.scss',
            'xamltech_addons_store/static/src/scss/apk.scss',
            'xamltech_addons_store/static/src/scss/blog.scss',
            'xamltech_addons_store/static/src/js/store_widget.js',
            'xamltech_addons_store/static/src/js/blog_animations.js',
        ],
    },
    'post_init_hook': '_post_init_hook',
    'installable': True,
    'application': True,
    'auto_install': False,
}
