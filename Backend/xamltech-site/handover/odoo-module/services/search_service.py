# -*- coding: utf-8 -*-
"""
Search Service — Query Building Layer
======================================
Encapsulates ORM domain and order logic. Pure functions, no state.
Single Responsibility: translate user input → ORM filters.
"""
from typing import Optional, List
from ..constants import AddonSort, AddonState


class SearchService:
    """Single Responsibility: build ORM queries for addon search"""

    BASE_DOMAIN = [
        ('state', '=', AddonState.PUBLISHED),
        ('website_published', '=', True),
    ]

    @staticmethod
    def build_domain(category: Optional[str] = None,
                     search: Optional[str] = None) -> List:
        """
        Build ORM domain filter from user inputs.
        Handles category filter and full-text search.
        Args:
            category: category name to filter (None = all)
            search: search term across name, description, tags
        Returns:
            ORM domain filter list
        """
        domain = SearchService.BASE_DOMAIN.copy()

        if category and category != 'All':
            domain.append(('category_id.name', '=', category))

        if search:
            search_domain = [
                '|', '|',
                ('name', 'ilike', search),
                ('description', 'ilike', search),
                ('tag_ids.name', 'ilike', search),
            ]
            domain.extend(search_domain)

        return domain

    @staticmethod
    def resolve_order(sort: str = AddonSort.POPULAR) -> str:
        """
        Resolve sort parameter to ORM order string.
        Validates against allowed sorts (prevents injection).
        Args:
            sort: sort option from AddonSort constants
        Returns:
            ORM order clause (e.g. 'downloads_count desc')
        """
        return AddonSort.ORM_MAP.get(sort, AddonSort.ORM_MAP[AddonSort.POPULAR])
