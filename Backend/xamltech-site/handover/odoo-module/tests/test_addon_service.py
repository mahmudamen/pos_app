# -*- coding: utf-8 -*-
"""
Test Addon Service — TDD Unit Tests
====================================
Tests for addon business logic and serialization.
"""
from odoo.tests import TransactionCase, tagged
from ..services.addon_service import AddonService


@tagged('xamltech_store', 'addon_service')
class TestAddonService(TransactionCase):
    """Test addon domain operations and serialization"""

    @classmethod
    def setUpClass(cls):
        super().setUpClass()
        cls.category = cls.env['xamltech.addon.category'].create({
            'name': 'Test Category',
            'icon_emoji': '📦',
        })
        cls.addon = cls.env['xamltech.addon'].create({
            'name': 'Test Addon',
            'slug': 'test-addon',
            'category_id': cls.category.id,
            'description': 'Test addon description for testing purposes',
            'version': '19.0.1.0.0',
            'state': 'published',
            'website_published': True,
            'downloads_count': 5,
            'rating': 4.5,
            'reviews_count': 10,
        })
        cls.addon_draft = cls.env['xamltech.addon'].create({
            'name': 'Draft Addon',
            'slug': 'draft-addon',
            'category_id': cls.category.id,
            'description': 'Draft addon should not appear',
            'version': '19.0.1.0.0',
            'state': 'draft',
            'website_published': False,
        })

    def test_get_published_returns_only_published(self):
        service = AddonService(self.env)
        published = service.get_published()
        slugs = published.mapped('slug')
        self.assertIn('test-addon', slugs)
        self.assertNotIn('draft-addon', slugs)

    def test_get_by_slug_returns_correct_addon(self):
        service = AddonService(self.env)
        addon = service.get_by_slug('test-addon')
        self.assertEqual(len(addon), 1)
        self.assertEqual(addon.name, 'Test Addon')

    def test_get_by_slug_returns_empty_for_unknown(self):
        service = AddonService(self.env)
        addon = service.get_by_slug('nonexistent-addon')
        self.assertEqual(len(addon), 0)

    def test_get_by_slug_excludes_draft(self):
        service = AddonService(self.env)
        addon = service.get_by_slug('draft-addon')
        self.assertEqual(len(addon), 0)

    def test_get_related_returns_same_category(self):
        addon2 = self.env['xamltech.addon'].create({
            'name': 'Related Addon',
            'slug': 'related-addon',
            'category_id': self.category.id,
            'description': 'Related addon in same category',
            'version': '19.0.1.0.0',
            'state': 'published',
            'website_published': True,
        })
        service = AddonService(self.env)
        related = service.get_related(self.addon)
        self.assertIn(addon2.id, related.ids)

    def test_get_related_excludes_self(self):
        service = AddonService(self.env)
        related = service.get_related(self.addon)
        self.assertNotIn(self.addon.id, related.ids)

    def test_increment_download_increases_count(self):
        service = AddonService(self.env)
        initial_count = self.addon.downloads_count
        ok = service.increment_download(self.addon.id)
        self.assertTrue(ok)
        self.addon.refresh()
        self.assertEqual(self.addon.downloads_count, initial_count + 1)

    def test_increment_download_returns_false_for_nonexistent(self):
        service = AddonService(self.env)
        ok = service.increment_download(999999)
        self.assertFalse(ok)

    def test_count_returns_published_only(self):
        service = AddonService(self.env)
        count = service.count()
        self.assertEqual(count, 1)

    def test_get_store_stats_calculates_correctly(self):
        addon2 = self.env['xamltech.addon'].create({
            'name': 'Addon 2',
            'slug': 'addon-2',
            'category_id': self.category.id,
            'description': 'Second addon for stats',
            'version': '19.0.1.0.0',
            'state': 'published',
            'website_published': True,
            'downloads_count': 10,
            'rating': 5.0,
            'reviews_count': 5,
        })
        service = AddonService(self.env)
        addons = service.get_published()
        stats = service.get_store_stats(addons)
        self.assertEqual(stats['total_addons'], 2)
        self.assertEqual(stats['total_downloads'], 15)
        self.assertEqual(stats['total_reviews'], 15)

    def test_serialize_returns_all_required_keys(self):
        service = AddonService(self.env)
        data = service.serialize(self.addon)
        required_keys = [
            'id', 'name', 'slug', 'category', 'description', 'price',
            'rating', 'tags', 'seoMetaDescription', 'seoKeywords', 'seoOgImage',
        ]
        for key in required_keys:
            self.assertIn(key, data)

    def test_serialize_handles_json_compat_versions(self):
        service = AddonService(self.env)
        data = service.serialize(self.addon)
        self.assertIn('compatibleVersions', data)
        self.assertIsInstance(data['compatibleVersions'], list)
