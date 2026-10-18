# -*- coding: utf-8 -*-
"""
Main Controller — Re-export for Backward Compatibility
======================================================
Imports and re-exports page + API controllers.
This preserves old routes while delegating to new service-based implementation.
"""
from .pages import XamltechPageController
from .api import XamltechApiController

__all__ = ['XamltechPageController', 'XamltechApiController','XamlTechVerification']
