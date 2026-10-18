# -*- coding: utf-8 -*-
"""
Addon Service — Domain Logic Layer
===================================
Encapsulates all addon business logic. Single Responsibility Principle:
queries, serialization, and statistics only.
No HTTP controllers logic mixed in.
"""
import json
from typing import Optional, List, Dict, Any
from ..constants import AddonSort, AddonState


class AddonService:
    """Single Responsibility: manage addon domain operations"""

    def __init__(self, env):
        """Inject Odoo environment (Dependency Inversion)"""
        self._env = env
        self._model = env['xamltech.addon']

    def get_published(self, domain: Optional[List] = None,
                      order: str = 'downloads_count desc',
                      limit: int = 12, offset: int = 0) -> Any:
        """
        Retrieve published addons with optional domain filter.
        Args:
            domain: ORM domain filter (default: published only)
            order: ORM order string
            limit: result limit
            offset: result offset for pagination
        Returns:
            Recordset of xamltech.addon
        """
        if domain is None:
            domain = self._get_published_domain()
        return self._model.sudo().search(domain, order=order, limit=limit, offset=offset)

    def get_by_slug(self, slug: str) -> Any:
        """
        Get addon by technical slug.
        Args:
            slug: addon technical name
        Returns:
            Single addon record or empty recordset
        """
        domain = [('slug', '=', slug), ('state', '=', AddonState.PUBLISHED)]
        return self._model.sudo().search(domain, limit=1)

    def get_related(self, addon: Any, limit: int = 3) -> Any:
        """
        Get related addons from same category (excluding self).
        Args:
            addon: xamltech.addon record
            limit: max results
        Returns:
            Recordset of related addons
        """
        addon.ensure_one()
        domain = [
            ('category_id', '=', addon.category_id.id),
            ('id', '!=', addon.id),
            ('state', '=', AddonState.PUBLISHED),
        ]
        return self._model.sudo().search(domain, limit=limit)

    def increment_download(self, addon_id: int) -> bool:
        """
        Increment addon download counter (for tracking/analytics).
        Args:
            addon_id: database ID of addon
        Returns:
            True if successful
        """
        try:
            addon = self._model.sudo().browse(addon_id)
            if not addon.exists():
                return False
            addon.write({'downloads_count': addon.downloads_count + 1})
            return True
        except Exception:
            return False

    def count(self, domain: Optional[List] = None) -> int:
        """
        Count published addons matching domain.
        Args:
            domain: ORM domain filter
        Returns:
            Count of matching records
        """
        if domain is None:
            domain = self._get_published_domain()
        return self._model.sudo().search_count(domain)

    def get_store_stats(self, addons: Any) -> Dict[str, Any]:
        """
        Calculate store statistics from addon recordset.
        Args:
            addons: xamltech.addon recordset
        Returns:
            dict with total_addons, total_downloads, avg_rating, total_reviews
        """
        addon_count = len(addons)
        if addon_count == 0:
            return {
                'total_addons': 0,
                'total_downloads': 0,
                'avg_rating': 0.0,
                'total_reviews': 0,
            }
        return {
            'total_addons': addon_count,
            'total_downloads': sum(addons.mapped('downloads_count')),
            'avg_rating': round(sum(addons.mapped('rating')) / addon_count, 1),
            'total_reviews': sum(addons.mapped('reviews_count')),
        }

    def serialize(self, addon: Any) -> Dict[str, Any]:
        """
        Serialize addon to JSON-safe dict for frontend/API.
        Single method responsibility: format for consumption only.
        Args:
            addon: xamltech.addon record
        Returns:
            dict with all addon data
        """
        addon.ensure_one()
        try:
            compat = json.loads(addon.compatible_versions or '[]')
        except (json.JSONDecodeError, TypeError):
            compat = []

        return {
            'id': addon.id,
            'name': addon.name,
            'slug': addon.slug,
            'category': addon.category_id.name,
            'categoryIcon': addon.category_id.icon_emoji or '📁',
            'description': addon.description,
            'fullDescription': addon.full_description or '',
            'iconEmoji': addon.icon_emoji,
            'accentColor': addon.accent_color,
            'price': addon.price,
            'version': addon.version,
            'compatibleVersions': compat,
            'rating': addon.rating,
            'reviewsCount': addon.reviews_count,
            'downloadsCount': addon.downloads_count,
            'badge': addon.badge_label or '',
            'badgeType': addon.badge or '',
            'isFeatured': addon.is_featured,
            'odooStoreUrl': addon.odoo_store_url or '',
            'githubUrl': addon.github_url or '',
            'docUrl': addon.documentation_url or '',
            'tags': addon.tag_ids.mapped('name'),
            'seoMetaDescription': addon.seo_meta_description or '',
            'seoKeywords': addon.seo_keywords or '',
            'seoOgImage': addon.seo_og_image or '',
        }

    def _get_published_domain(self) -> List:
        """
        Base domain for all published addon queries.
        Returns:
            ORM domain filter for published addons
        """
        return [
            ('state', '=', AddonState.PUBLISHED),
            ('website_published', '=', True),
        ]
