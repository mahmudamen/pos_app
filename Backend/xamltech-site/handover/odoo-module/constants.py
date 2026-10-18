# -*- coding: utf-8 -*-
"""
XamlTech Addons Store Constants
================================
Enums, sort maps, error messages, and configuration constants.
Used across services, validators, and controllers.
"""


class AddonSort:
    """Sort options for addon listings"""
    POPULAR = 'popular'
    RATING = 'rating'
    NEWEST = 'newest'
    PRICE_LOW = 'price_low'
    PRICE_HIGH = 'price_high'

    ORM_MAP = {
        POPULAR: 'downloads_count desc',
        RATING: 'rating desc',
        NEWEST: 'create_date desc',
        PRICE_LOW: 'price asc',
        PRICE_HIGH: 'price desc',
    }

    CHOICES = [
        (POPULAR, 'Most Popular'),
        (RATING, 'Highest Rated'),
        (NEWEST, 'Newest First'),
        (PRICE_LOW, 'Price: Low to High'),
        (PRICE_HIGH, 'Price: High to Low'),
    ]


class AddonState:
    """Addon publication states"""
    DRAFT = 'draft'
    PUBLISHED = 'published'
    ARCHIVED = 'archived'

    CHOICES = [
        (DRAFT, 'Draft'),
        (PUBLISHED, 'Published'),
        (ARCHIVED, 'Archived'),
    ]


class AddonBadge:
    """Addon distinction badges"""
    BEST_SELLER = 'best_seller'
    POPULAR = 'popular'
    ESSENTIAL = 'essential'
    TOP_RATED = 'top_rated'
    NEW = 'new'
    ADVANCED = 'advanced'

    CHOICES = [
        (BEST_SELLER, 'Best Seller'),
        (POPULAR, 'Popular'),
        (ESSENTIAL, 'Essential'),
        (TOP_RATED, 'Top Rated'),
        (NEW, 'New'),
        (ADVANCED, 'Advanced'),
    ]


class ErrorMessage:
    """User-facing error messages"""
    # Contact validation
    INVALID_NAME = "Name must be between 2 and 100 characters."
    INVALID_EMAIL = "Please enter a valid email address."
    INVALID_MESSAGE = "Message must be at least 10 characters."
    INVALID_PHONE = "Phone must be 7-20 digits with optional + prefix."
    INVALID_SUBJECT = "Subject is required."

    # Demo validation
    INVALID_WHATSAPP = "WhatsApp number must be 7-20 digits with optional + prefix."

    # Addon operations
    ADDON_NOT_FOUND = "Addon not found or is not published."
    INVALID_ADDON_ID = "Invalid addon ID."

    # CRM operations
    CRM_TEAM_NOT_FOUND = "CRM team 'Demo Requests' not found."
    CRM_CREATE_FAILED = "Failed to create CRM lead. Please try again."


class ValidationRegex:
    """Regular expressions for validation"""
    EMAIL = r'^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$'
    PHONE = r'^\+?[0-9]{7,20}$'
    # Supports Latin, Arabic Unicode block (U+0600-U+06FF), spaces, hyphens, apostrophes
    NAME = r'^[\w\s\-\'\.À-öø-ÿ؀-ۿݐ-ݿ]+$'


class AppConfig:
    """Application configuration"""
    CONTACT_FORM_MIN_NAME = 2
    CONTACT_FORM_MAX_NAME = 100
    CONTACT_FORM_MIN_MESSAGE = 10
    DEMO_FORM_MIN_NAME = 2
    DEMO_FORM_MAX_NAME = 100

    ADDON_STORE_PAGE_SIZE = 12
    ADDON_SEARCH_LIMIT = 50

    CRM_TEAM_NAME = 'Demo Requests'
    CRM_DEMO_TAG_NAME = 'Demo Request'
    CRM_APK_TAG_NAME = 'APK Tester'
    CRM_CONTACT_TAG_NAME = 'Website Contact'
    DEMO_URL = 'https://demo18.xamltech.com'
    DEMO_CREDENTIALS = 'demo / demo'
