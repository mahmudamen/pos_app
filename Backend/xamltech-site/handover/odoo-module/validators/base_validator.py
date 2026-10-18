# -*- coding: utf-8 -*-
"""
Base Validator — OCP Foundation
================================
Abstract base for all validators. Extend to create new validators.
Open/Closed Principle: new validators add functionality without modifying base.
"""
from abc import ABC, abstractmethod
from typing import Dict


class ValidationResult:
    """Encapsulates validation outcome with error dict"""

    def __init__(self):
        self.errors: Dict[str, str] = {}

    @property
    def is_valid(self) -> bool:
        """True if no errors recorded"""
        return not self.errors

    def add_error(self, field: str, message: str):
        """Record a validation error for a field"""
        self.errors[field] = message


class BaseValidator(ABC):
    """Abstract base for all form validators (OCP)"""

    @abstractmethod
    def validate(self, data: dict) -> ValidationResult:
        """
        Validate input dict. Override in subclasses.
        Args:
            data: input fields dict
        Returns:
            ValidationResult with errors dict populated if invalid
        """
        pass
