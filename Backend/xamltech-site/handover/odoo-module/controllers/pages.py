# -*- coding: utf-8 -*-
"""
Pages Controller — Static Page Routes
======================================
Thin controllers. Each route: render template. No business logic.
Single Responsibility: page rendering only.
"""
from odoo import http
from odoo.http import request


class XamltechPageController(http.Controller):
    """Thin page routes — render templates only"""

    @http.route('/contact-us', type='http', auth='public', website=True, sitemap=True)
    def contact_page(self, **kw):
        return request.render('xamltech_addons_store.contact_page')

    @http.route('/request-demo', type='http', auth='public', website=True, sitemap=True)
    def demo_page(self, **kw):
        return request.render('xamltech_addons_store.demo_page')

    @http.route('/solutions/erp', type='http', auth='public', website=True, sitemap=True)
    def erp_page(self, **kw):
        return request.render('xamltech_addons_store.erp_page')

    @http.route('/solutions/it-infrastructure', type='http', auth='public', website=True, sitemap=True)
    def infra_page(self, **kw):
        return request.render('xamltech_addons_store.infra_page')

    @http.route('/solutions/web', type='http', auth='public', website=True, sitemap=True)
    def web_dev_page(self, **kw):
        return request.render('xamltech_addons_store.web_dev_page')

    @http.route('/portfolio', type='http', auth='public', website=True, sitemap=True)
    def portfolio_page(self, **kw):
        return request.render('xamltech_addons_store.portfolio_page')

    @http.route('/knowledge', type='http', auth='public', website=True, sitemap=True)
    def knowledge_page(self, **kw):
        return request.render('xamltech_addons_store.knowledge_page')

    @http.route('/odoo-pos-egypt', type='http', auth='public', website=True, sitemap=True)
    def pos_page(self, **kw):
        return request.render('xamltech_addons_store.pos_page')

    @http.route('/erp-agriculture-egypt', type='http', auth='public', website=True, sitemap=True)
    def agri_page(self, **kw):
        return request.render('xamltech_addons_store.agri_page')

    @http.route('/theme-info', type='http', auth='public', website=True, sitemap=True)
    def theme_info_page(self, **kw):
        return request.render('xamltech_addons_store.xt_theme_info_page')

    @http.route('/ai-solutions', type='http', auth='public', website=True, sitemap=True)
    def ai_solutions_page(self, **kw):
        return request.render('xamltech_addons_store.ai_apps_page')

    @http.route('/testimonials', type='http', auth='public', website=True, sitemap=True)
    def testimonials_page(self, **kw):
        return request.render('xamltech_addons_store.testimonials_page')

    @http.route('/pricing', type='http', auth='public', website=True, sitemap=True)
    def pricing_page(self, **kw):
        return request.render('xamltech_addons_store.pricing_page')

    @http.route('/networktool', type='http', auth='public', website=True, sitemap=True)
    def networktool_page(self, **kw):
        return request.render('xamltech_addons_store.networktool_page')

    @http.route('/insights', type='http', auth='public', website=True, sitemap=True)
    def insights_hub_page(self, **kw):
        return request.render('xamltech_addons_store.blog_hub_page')

    @http.route('/insights/ai-marketing-copilot', type='http', auth='public', website=True, sitemap=True)
    def insights_ai_marketing(self, **kw):
        return request.render('xamltech_addons_store.blog_ai_marketing_copilot')

    @http.route('/insights/ai-pos-copilot', type='http', auth='public', website=True, sitemap=True)
    def insights_ai_pos(self, **kw):
        return request.render('xamltech_addons_store.blog_ai_pos_copilot')

    # ── AISEO: LLM-readable site summaries ───────────────────────

    @http.route('/llms.txt', type='http', auth='public', website=True, sitemap=False)
    def llms_txt(self, **kw):
        content = """# XamlTech — Odoo ERP, AI & IT Solutions
> XamlTech is a full-service IT company in Aswan, Egypt specializing in Odoo ERP (v16-19), AI-powered modules, networking, CCTV, and web development.

## Company
- Founded: 2018 by Mahmoud Amen & Osama M Salah
- Location: Aswan, Egypt
- Contact: mahmudamen@gmail.com | +20 100 798 2468
- Website: https://xamltech.com

## Core Services
- **Odoo ERP Implementation**: Custom modules for agriculture, textile, healthcare, manufacturing. Versions 16-19.
- **AI Copilots for Odoo**: Marketing Copilot (campaign optimization, Thompson Sampling) and POS Copilot (real-time upsell suggestions, multi-LLM).
- **POS Customization**: Published modules on Odoo Apps Store — cutting orders for textile, real-time stock sync, security framework.
- **IT Infrastructure**: Enterprise LAN/WAN, CCTV surveillance (up to 128 channels), Linux servers, Proxmox/Docker.
- **Web Development**: Responsive websites, Odoo themes, PWA, SEO optimization.

## Published Odoo Modules (Odoo Apps Store)
- POS Cutting Orders (ma_pos_cutting) — €149 — Fabric cutting management for textile POS
- POS Stock Real-Time (ma_rs_pos_stock) — €129 — Live stock sync across POS sessions
- POS Security Framework (ma_pos_base) — €99 — Manager PIN, discount limits, session controls
- Agricultural Management (agricultural_management) — Farm hierarchy, crop tracking, harvest analytics

## Key Projects
- Hospital ERP: Full Odoo deployment for Furkan Surgical Center (patient records, pharmacy, billing)
- Hospital CCTV: 128-channel IP camera system for Furkan Surgical Center
- Textile POS: Custom Odoo 18 POS for Al-Ghazl Al-Arabi Trading Company
- ZATCA E-Invoicing: Saudi Arabia Phase 2 compliance
- Farm Management: Odoo ERP for Qatf Company agriculture operations

## Technologies
Python, JavaScript, OWL v2/v3, PostgreSQL, Odoo (v12-v19), Docker, Proxmox, Linux, Chart.js

## Pages
- /solutions/erp — Odoo ERP development services
- /solutions/it-infrastructure — Networking and CCTV services
- /solutions/web — Web development services
- /ai-solutions — AI Marketing and POS Copilot modules
- /addons-store — Browse and purchase Odoo modules
- /portfolio — Completed project showcase
- /insights — Technical articles and guides
- /pricing — Service and module pricing with ROI calculator
- /contact-us — Get in touch
- /request-demo — Request a free demo
- /networktool — NetworkTool Android app: 40+ network diagnostics, WiFi analyzer, AI Copilot, screenshots and tester program
- /privacy-policy — NetworkTool privacy policy
"""
        return request.make_response(
            content,
            headers=[
                ('Content-Type', 'text/plain; charset=utf-8'),
                ('Cache-Control', 'public, max-age=86400'),
            ],
        )

    @http.route('/llms-full.txt', type='http', auth='public', website=True, sitemap=False)
    def llms_full_txt(self, **kw):
        content = """# XamlTech — Odoo ERP, AI & IT Solutions (Full Reference)
> XamlTech is a full-service IT company in Aswan, Egypt specializing in Odoo ERP (v16-19), AI-powered modules, networking, CCTV, and web development. We serve clients across Egypt, Saudi Arabia, and the MENA region with enterprise-grade solutions.

## Company
- Founded: 2018 by Mahmoud Amen & Osama M Salah
- Location: Aswan, Egypt (serving all of Egypt, Saudi Arabia, and MENA)
- Contact: mahmudamen@gmail.com | +20 100 798 2468
- Website: https://xamltech.com
- GitHub: https://github.com/XamlTech
- Odoo Apps Publisher: https://apps.odoo.com/apps/modules/browse?author=Mahmoud%20Amen

## Core Services

### Odoo ERP Implementation
Custom modules for agriculture, textile, healthcare, manufacturing. Versions 16-19.
We handle the full lifecycle: requirements analysis, module development, data migration from legacy systems (Oracle, Excel), user training in Arabic, and ongoing support. Our team has 10+ years of Odoo experience spanning versions 12 through 19, with deep expertise in Arabic RTL localization and Egyptian government compliance (tax, social insurance, civil service law 81/2016).

### AI Copilots for Odoo
Marketing Copilot (campaign optimization, Thompson Sampling) and POS Copilot (real-time upsell suggestions, multi-LLM).
The Marketing Copilot uses multi-armed bandit algorithms (Thompson Sampling) to automatically optimize campaign budget allocation across channels, learning from real conversion data. The POS Copilot provides real-time product recommendations at checkout using a multi-LLM architecture — it analyzes basket contents, customer history, and inventory levels to suggest upsells and cross-sells with configurable confidence thresholds.

### POS Customization
Published modules on Odoo Apps Store — cutting orders for textile, real-time stock sync, security framework.
Our POS modules are production-tested in real retail environments. The Cutting Orders module was built for textile businesses that need to track fabric cutting operations directly from the POS interface. The Stock Real-Time module eliminates the stock sync delay that causes overselling in multi-session environments. The Security Framework adds manager PIN verification, discount percentage limits, and session access controls.

### IT Infrastructure
Enterprise LAN/WAN, CCTV surveillance (up to 128 channels), Linux servers, Proxmox/Docker.
We design and deploy complete network infrastructure for hospitals, offices, and commercial facilities. Our CCTV installations range from small 8-camera retail setups to enterprise 128-channel hospital systems with NVR recording, mobile app remote monitoring, and integration with access control systems. Server infrastructure includes Proxmox virtualization clusters, Docker containerization, automated backups, and 24/7 monitoring.

### Web Development
Responsive websites, Odoo themes, PWA, SEO optimization.
We build modern, fast websites using the Odoo website framework with custom themes, Progressive Web App (PWA) capabilities for offline access, and comprehensive SEO including structured data (JSON-LD), meta optimization, and performance tuning. All sites are Arabic RTL-first with bilingual support.

## Published Odoo Modules (Odoo Apps Store)
- POS Cutting Orders (ma_pos_cutting) — €149 — Fabric cutting management for textile POS. Manage cutting orders directly from POS, track fabric rolls, calculate waste, generate cutting tickets, dark/light mode UI.
- POS Stock Real-Time (ma_rs_pos_stock) — €129 — Live stock sync across POS sessions. Eliminates overselling with real-time quantity updates, multi-warehouse support, configurable sync intervals.
- POS Security Framework (ma_pos_base) — €99 — Manager PIN, discount limits, session controls. Role-based access, PIN verification for sensitive operations, maximum discount enforcement, session audit trail.
- Agricultural Management (agricultural_management) — Farm hierarchy, crop tracking, harvest analytics. Hierarchical farm/plot/block structure, crop lifecycle management, harvest recording with quality grading, weather integration, analytic accounting per farm unit.

## Key Projects
- Hospital ERP: Full Odoo deployment for Furkan Surgical Center — patient records, appointment scheduling, pharmacy with barcode scanning, billing and insurance claims, Arabic reports, integrated with CCTV and network infrastructure.
- Hospital CCTV: 128-channel IP camera system for Furkan Surgical Center — operating rooms, corridors, pharmacy, emergency entrance, parking lot, 24/7 NVR recording with 30-day retention, mobile app monitoring for administration.
- Textile POS: Custom Odoo 18 POS for Al-Ghazl Al-Arabi Trading Company — cutting order management, fabric roll tracking, real-time stock sync across 3 POS sessions, manager security controls, daily sales and cutting reports.
- ZATCA E-Invoicing: Saudi Arabia Phase 2 compliance — XML invoice generation per ZATCA specifications, QR code embedding, cryptographic signing, API submission to Fatoora portal, reconciliation dashboard.
- Farm Management: Odoo ERP for Qatf Company agriculture operations — hierarchical farm locations (farm > plot > block), crop tracking with growth stages, harvest recording, equipment maintenance scheduling, cost analysis per crop cycle.
- Government HR System: Comprehensive HR and payroll system for Egyptian government authority (High Dam Authority, Aswan) — full compliance with Civil Service Law 81/2016, social insurance Law 148/2019, graduated tax brackets, 9 integrated modules covering HR, payroll, leaves, medical claims, settlements, training, and document management.

## Technologies
Python, JavaScript, OWL v2/v3, PostgreSQL, Odoo (v12-v19), Docker, Proxmox, Linux, Chart.js, SCSS/CSS custom properties, PWA/Service Workers, JSON-LD structured data

## Pages
- /solutions/erp — Odoo ERP development services: custom modules, implementation methodology, published addons, pricing packages
- /solutions/it-infrastructure — Networking and CCTV services: LAN/WAN design, structured cabling, CCTV surveillance, server deployment
- /solutions/web — Web development services: responsive sites, Odoo themes, PWA, SEO optimization
- /ai-solutions — AI Marketing and POS Copilot modules: Thompson Sampling optimization, multi-LLM recommendations
- /addons-store — Browse and purchase Odoo modules: POS Cutting Orders, Stock Real-Time, Security Framework, Agriculture
- /portfolio — Completed project showcase: hospitals, textile, agriculture, government, ZATCA
- /insights — Technical articles and guides: AI copilot deep-dives, Odoo development tutorials
- /pricing — Service and module pricing with interactive ROI calculator
- /contact-us — Get in touch: contact form, WhatsApp, email, phone
- /request-demo — Request a free live demo of any module or service
- /knowledge — Knowledge base: Odoo development best practices, technical documentation
- /testimonials — Customer success stories and case studies
- /odoo-pos-egypt — Dedicated POS solutions page for Egyptian retail market
- /erp-agriculture-egypt — Agriculture ERP solutions for Egyptian farms
- /networktool — NetworkTool Android app: 40+ network diagnostics, WiFi analyzer, AI Copilot, screenshots and tester program
- /privacy-policy — NetworkTool privacy policy
"""
        return request.make_response(
            content,
            headers=[
                ('Content-Type', 'text/plain; charset=utf-8'),
                ('Cache-Control', 'public, max-age=86400'),
            ],
        )
