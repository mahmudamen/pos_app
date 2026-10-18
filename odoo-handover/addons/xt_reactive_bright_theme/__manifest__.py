{
    "name": "Reactive Bright Theme",
    "version": "19.0.1.0.0",
    "category": "Themes/Backend",
    "summary": "Calm modern backend and login theme inspired by React-style interfaces.",
    "description": """
Reactive Bright Theme
=====================

A CSS-only Odoo 19 theme with a calm modern interface:

- Balanced teal, indigo, slate, and soft status colors.
- Cleaner navbar, control panel, forms, lists, kanban cards, dialogs, and buttons.
- Login styling that also improves the existing custom login page when installed.
- Safe CSS asset overrides only.
    """,
    "author": "Xaml Tech",
    "license": "LGPL-3",
    "depends": ["web"],
    "assets": {
        "web.assets_backend": [
            "xt_reactive_bright_theme/static/src/scss/reactive_bright_theme.scss",
        ],
        "web.assets_web": [
            "xt_reactive_bright_theme/static/src/scss/reactive_bright_theme.scss",
        ],
    },
    "data": [
        "data/assets.xml",
    ],
    "installable": True,
    "application": True,
    "auto_install": False,
}
