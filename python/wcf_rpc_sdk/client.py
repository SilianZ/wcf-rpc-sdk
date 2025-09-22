"""High level client implementation for the Python WCF RPC SDK."""
from __future__ import annotations

import logging
import threading
from typing import Callable, List, Optional

from .exceptions import BufferFullError, DependencyMissingError
from .message import (
    Contact,
    Message,
    MessageBuffer,
    SelfInfo,
    contact_from_proto,
    message_from_proto,
)
from .transport import NanomsgTransport
from .utils import is_chatroom_type, is_friend_type, is_gh_type

_logger = logging.getLogger(__name__)


class _SelfProfile:
    def __init__(self, transport: NanomsgTransport) -> None:
        self._transport = transport
        self._info: Optional[SelfInfo] = None
        self._contacts: dict[str, Contact] = {}
        self._lock = threading.RLock()

    def refresh_info(self) -> Optional[SelfInfo]:
        proto = self._transport.get_user_info()
        if proto is None:
            return None
        info = SelfInfo(
            wxid=proto.wxid,
            name=proto.name,
            mobile=proto.mobile,
            home=proto.home,
            file_storage_path=f"{proto.home.rstrip('/')}/{proto.wxid}/FileStorage",
        )
        with self._lock:
            self._info = info
        return info

    def get_info(self, refresh: bool = False) -> Optional[SelfInfo]:
        with self._lock:
            if self._info is not None and not refresh:
                return self._info
        return self.refresh_info()

    def refresh_contacts(self) -> List[Contact]:
        contacts = [contact_from_proto(c) for c in self._transport.get_contacts()]
        for contact in contacts:
            if is_friend_type(contact.wxid):
                contact.flags |= contact.flags.__class__.FRIEND
            elif is_chatroom_type(contact.wxid):
                contact.flags |= contact.flags.__class__.CHATROOM
            elif is_gh_type(contact.wxid):
                contact.flags |= contact.flags.__class__.OFFICIAL
        with self._lock:
            self._contacts = {c.wxid: c for c in contacts}
        return contacts

    def get_contacts(self, refresh: bool = False) -> List[Contact]:
        with self._lock:
            if self._contacts and not refresh:
                return list(self._contacts.values())
        return self.refresh_contacts()

    def get_friends(self) -> List[Contact]:
        return [c for c in self.get_contacts() if is_friend_type(c.wxid)]

    def get_chatrooms(self) -> List[Contact]:
        return [c for c in self.get_contacts() if is_chatroom_type(c.wxid)]


class Client:
    def __init__(
        self,
        msg_buffer_size: int = 10,
        *,
        address: Optional[str] = None,
        transport_factory: Callable[[Optional[str]], NanomsgTransport] = NanomsgTransport,
    ) -> None:
        try:
            self._transport = transport_factory(address)
        except DependencyMissingError:
            raise
        self._msg_buffer = MessageBuffer(msg_buffer_size)
        self._stop_event = threading.Event()
        self._listener_thread: Optional[threading.Thread] = None
        self._self = _SelfProfile(self._transport)

    def run(self) -> None:
        status = self._transport.enable_recv_text()
        if status != 1:
            _logger.warning("failed to enable message reception: status=%s", status)
            return
        if self._listener_thread and self._listener_thread.is_alive():
            return
        self._stop_event.clear()
        self._listener_thread = threading.Thread(
            target=self._listen_loop,
            name="wcf-message-listener",
            daemon=True,
        )
        self._listener_thread.start()

    def _listen_loop(self) -> None:
        def on_message(raw_msg) -> None:
            message = message_from_proto(raw_msg)
            message.client = self
            try:
                self._msg_buffer.put(message)
            except BufferFullError:
                _logger.warning("message buffer full; dropping message id=%s", raw_msg.id)

        try:
            self._transport.listen(on_message, self._stop_event)
        except DependencyMissingError as exc:  # pragma: no cover
            _logger.error("Cannot receive messages because pynng is missing: %s", exc)
        finally:
            self._msg_buffer.close()

    def close(self) -> None:
        self._stop_event.set()
        if self._listener_thread and self._listener_thread.is_alive():
            self._listener_thread.join(timeout=2)
        try:
            self._transport.disable_recv_text()
        except Exception:  # pragma: no cover
            pass
        self._transport.close()
        self._msg_buffer.close()

    def get_message(self, timeout: Optional[float] = None) -> Message:
        return self._msg_buffer.get(timeout=timeout)

    def get_message_nowait(self) -> Message:
        return self._msg_buffer.get(timeout=0)

    def is_login(self) -> bool:
        return self._transport.is_login()

    def send_text(self, receiver: str, content: str, *ats: str) -> None:
        status = self._transport.send_text(receiver, content, ats)
        if status != 0:
            raise RuntimeError(f"SendTxt failed with status {status}")

    def send_image(self, receiver: str, path: str) -> None:
        status = self._transport.send_image(receiver, path)
        if status != 0:
            raise RuntimeError(f"SendIMG failed with status {status}")

    def send_file(self, receiver: str, path: str) -> None:
        status = self._transport.send_file(receiver, path)
        if status != 0:
            raise RuntimeError(f"SendFile failed with status {status}")

    def send_xml(self, receiver: str, content: str, *, path: str = "", msg_type: int = 0) -> None:
        status = self._transport.send_xml(receiver, content, path=path, msg_type=msg_type)
        if status != 0:
            raise RuntimeError(f"SendXml failed with status {status}")

    def send_card(self, receiver: str, **kwargs: str) -> None:
        status = self._transport.send_rich_text(receiver, **kwargs)
        if status != 1:
            raise RuntimeError(f"SendRichText failed with status {status}")

    def accept_new_friend(self, request: Message) -> bool:
        if not request.new_friend_request:
            return False
        req = request.new_friend_request
        status = self._transport.accept_friend(req.v3, req.v4, req.scene)
        return status == 1

    def get_self_info(self, refresh: bool = False) -> Optional[SelfInfo]:
        return self._self.get_info(refresh=refresh)

    def get_contacts(self, refresh: bool = False) -> List[Contact]:
        return self._self.get_contacts(refresh=refresh)

    def get_all_friend(self) -> List[Contact]:
        return self._self.get_friends()

    def get_all_chat_room(self) -> List[Contact]:
        return self._self.get_chatrooms()


__all__ = ["Client"]
