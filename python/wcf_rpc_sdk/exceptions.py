"""Custom exceptions used by the Python WCF RPC SDK."""
from __future__ import annotations


class WcfRpcError(Exception):
    """Base class for SDK specific exceptions."""


class BufferFullError(WcfRpcError):
    """Raised when the internal message buffer is full."""


class NotLoggedInError(WcfRpcError):
    """Raised when an operation requires an authenticated session."""


class DependencyMissingError(WcfRpcError, ImportError):
    """Raised when an optional runtime dependency is missing."""


class TransportClosedError(WcfRpcError):
    """Raised when trying to use a closed transport."""


class TimeoutError(WcfRpcError):
    """Raised when waiting for an operation exceeds the timeout."""


__all__ = [
    "WcfRpcError",
    "BufferFullError",
    "NotLoggedInError",
    "DependencyMissingError",
    "TransportClosedError",
    "TimeoutError",
]
