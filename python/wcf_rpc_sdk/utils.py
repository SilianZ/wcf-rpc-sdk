"""Utility helpers shared across the SDK."""
from __future__ import annotations

import ipaddress
from typing import Iterable, Optional


DEFAULT_ADDRESS = "tcp://127.0.0.1:10086"


def normalize_address(address: Optional[str]) -> str:
    """Validate and normalise the TCP endpoint string used by WCF."""

    if not address:
        return DEFAULT_ADDRESS
    address = address.strip()
    if not address:
        return DEFAULT_ADDRESS
    if not address.startswith("tcp://"):
        raise ValueError(f"Unsupported address: {address!r}. Only tcp:// endpoints are supported.")
    host_port = address[len("tcp://") :]
    host, _, port = host_port.rpartition(":")
    if not host or not port:
        raise ValueError(f"Invalid tcp endpoint: {address!r}")
    try:
        ipaddress.ip_address(host)
    except ValueError:
        raise ValueError(f"Invalid IP address in endpoint: {host!r}") from None
    try:
        port_int = int(port)
    except ValueError as exc:  # pragma: no cover
        raise ValueError(f"Invalid port in endpoint: {address!r}") from exc
    if not 0 < port_int < 65536:
        raise ValueError(f"Port out of range: {port_int}")
    return address


def increment_port(address: str) -> str:
    normalized = normalize_address(address)
    host_port = normalized[len("tcp://") :]
    host, _, port = host_port.rpartition(":")
    next_port = int(port) + 1
    return f"tcp://{host}:{next_port}"


def is_friend_type(wxid: str) -> bool:
    return bool(wxid and wxid.startswith("wxid_"))


def is_chatroom_type(wxid: str) -> bool:
    return bool(wxid and wxid.endswith("@chatroom"))


def is_gh_type(wxid: str) -> bool:
    return bool(wxid and wxid.startswith("gh_"))


def join_aters(ats: Iterable[str]) -> str:
    return ",".join(ats)


__all__ = [
    "DEFAULT_ADDRESS",
    "normalize_address",
    "increment_port",
    "is_friend_type",
    "is_chatroom_type",
    "is_gh_type",
    "join_aters",
]
