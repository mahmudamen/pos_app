# -*- coding: utf-8 -*-
"""
Test Search Service — TDD Domain Building Tests
================================================
Tests for query building and sort resolution.
"""
from odoo.tests import TransactionCase, tagged
from ..services.search_service import SearchService
from ..constants import AddonSort


@tagged('xamltech_store', 'search_service')
class TestSearchService(TransactionCase):
    """Test domain building and sort resolution (pure static methods)"""

    def test_build_domain_returns_published_filter(self):
        domain = SearchService.build_domain()
        self.assertEqual(domain[0], ('state', '=', 'published'))
        self.assertEqual(domain[1], ('website_published', '=', True))

    def test_build_domain_with_category_filter(self):
        domain = SearchService.build_domain(category='POS')
        cat_filter = ('category_id.name', '=', 'POS')
        self.assertIn(cat_filter, domain)

    def test_build_domain_ignores_all_category(self):
        domain = SearchService.build_domain(category='All')
        cat_filter = ('category_id.name', '=', 'All')
        self.assertNotIn(cat_filter, domain)

    def test_build_domain_with_search_term(self):
        domain = SearchService.build_domain(search='test')
        search_found = False
        for item in domain:
            if isinstance(item, tuple) and 'ilike' in str(item):
                search_found = True
                break
        self.assertTrue(search_found)

    def test_build_domain_search_covers_name_description_tags(self):
        domain = SearchService.build_domain(search='addon')
        domain_str = str(domain)
        self.assertIn('name', domain_str)
        self.assertIn('description', domain_str)
        self.assertIn('tag_ids.name', domain_str)

    def test_resolve_order_popular(self):
        order = SearchService.resolve_order(AddonSort.POPULAR)
        self.assertEqual(order, 'downloads_count desc')

    def test_resolve_order_rating(self):
        order = SearchService.resolve_order(AddonSort.RATING)
        self.assertEqual(order, 'rating desc')

    def test_resolve_order_newest(self):
        order = SearchService.resolve_order(AddonSort.NEWEST)
        self.assertEqual(order, 'create_date desc')

    def test_resolve_order_price_low(self):
        order = SearchService.resolve_order(AddonSort.PRICE_LOW)
        self.assertEqual(order, 'price asc')

    def test_resolve_order_price_high(self):
        order = SearchService.resolve_order(AddonSort.PRICE_HIGH)
        self.assertEqual(order, 'price desc')

    def test_resolve_order_defaults_to_popular_on_invalid(self):
        order = SearchService.resolve_order('invalid_sort')
        self.assertEqual(order, 'downloads_count desc')

    def test_resolve_order_empty_string_defaults_to_popular(self):
        order = SearchService.resolve_order('')
        self.assertEqual(order, 'downloads_count desc')
