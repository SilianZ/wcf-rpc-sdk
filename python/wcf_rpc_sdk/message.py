"""High level data structures used by the Python SDK."""
from __future__ import annotations

import queue
import threading
from dataclasses import dataclass, field
from enum import IntEnum, IntFlag
from typing import List, Optional, Sequence, TYPE_CHECKING

from . import wcf_pb2
from .exceptions import BufferFullError, TimeoutError

if TYPE_CHECKING:  # pragma: no cover
    from .client import Client


class GenderType(IntEnum):
    UNKNOWN = 0
    MALE = 1
    FEMALE = 2


class MsgType(IntEnum):
    MOMENTS = 0
    TEXT = 1
    IMAGE = 3
    VOICE = 34
    FRIEND_CONFIRM = 37
    POSSIBLE_FRIEND = 40
    BUSINESS_CARD = 42
    VIDEO = 43
    ROCK_PAPER_SCISSORS = 47
    LOCATION = 48
    XML = 49
    XML_QUOTE = 4901
    XML_IMAGE = 4903
    XML_FILE = 4906
    XML_LINK = 4916
    VOIP = 50
    WECHAT_INIT = 51
    VOIP_NOTIFY = 52
    VOIP_INVITE = 53
    SHORT_VIDEO = 62
    RED_PACKET = 66
    SYS_NOTICE = 9999
    SYSTEM = 10000
    REVOKE = 10002
    SOGOU_EMOJI = 1048625
    RED_PACKET_COVER = 536936497
    VIDEO_CHANNEL_VIDEO = 754974769


class ContactFlags(IntFlag):
    NONE = 0
    FRIEND = 1 << 0
    CHATROOM = 1 << 1
    OFFICIAL = 1 << 2


@dataclass(slots=True)
class Contact:
    wxid: str
    code: str = ""
    remark: str = ""
    name: str = ""
    country: str = ""
    province: str = ""
    city: str = ""
    gender: GenderType = GenderType.UNKNOWN
    flags: ContactFlags = ContactFlags.NONE


@dataclass(slots=True)
class SelfInfo:
    wxid: str
    name: str
    mobile: str = ""
    home: str = ""
    file_storage_path: str = ""


@dataclass(slots=True)
class NewFriendRequest:
    v3: str
    v4: str
    scene: int


@dataclass(slots=True)
class Message:
    is_self: bool
    is_group: bool
    message_id: int
    type: MsgType
    ts: int
    room_id: str
    content: str
    wxid: str
    sign: str
    thumb: str
    extra: str
    xml: str
    raw: wcf_pb2.WxMsg = field(repr=False)
    client: Optional["Client"] = field(default=None, repr=False, compare=False)
    new_friend_request: Optional[NewFriendRequest] = None

    def reply_text(self, content: str, ats: Sequence[str] | None = None) -> None:
        if not self.client:
            raise RuntimeError("message is not bound to a client")
        self.client.send_text(self.reply_target, content, *(ats or ()))

    def reply_image(self, path: str) -> None:
        if not self.client:
            raise RuntimeError("message is not bound to a client")
        self.client.send_image(self.reply_target, path)

    def reply_file(self, path: str) -> None:
        if not self.client:
            raise RuntimeError("message is not bound to a client")
        self.client.send_file(self.reply_target, path)

    @property
    def reply_target(self) -> str:
        return self.room_id or self.wxid


class MessageBuffer:
    def __init__(self, size: int):
        if size <= 0:
            raise ValueError("size must be positive")
        self._queue: queue.Queue[Message] = queue.Queue(maxsize=size)
        self._closed = threading.Event()

    def close(self) -> None:
        self._closed.set()

    def put(self, message: Message, timeout: Optional[float] = None) -> None:
        if self._closed.is_set():
            raise BufferFullError("buffer closed")
        try:
            self._queue.put(message, timeout=timeout)
        except queue.Full as exc:  # pragma: no cover
            raise BufferFullError("message buffer is full") from exc

    def get(self, timeout: Optional[float] = None) -> Message:
        if self._closed.is_set():
            raise TimeoutError("buffer closed")
        try:
            return self._queue.get(timeout=timeout)
        except queue.Empty as exc:
            raise TimeoutError("timed out waiting for message") from exc

    def drain(self) -> List[Message]:
        items: List[Message] = []
        while True:
            try:
                items.append(self._queue.get_nowait())
            except queue.Empty:
                break
        return items


def contact_from_proto(raw: wcf_pb2.RpcContact) -> Contact:
    return Contact(
        wxid=raw.wxid,
        code=raw.code,
        remark=raw.remark,
        name=raw.name,
        country=raw.country,
        province=raw.province,
        city=raw.city,
        gender=GenderType(raw.gender),
    )


def message_from_proto(raw: wcf_pb2.WxMsg) -> Message:
    msg = Message(
        is_self=raw.is_self,
        is_group=raw.is_group,
        message_id=raw.id,
        type=MsgType(raw.type) if raw.type in MsgType._value2member_map_ else MsgType.TEXT,
        ts=raw.ts,
        room_id=raw.roomid,
        content=raw.content,
        wxid=raw.sender,
        sign=raw.sign,
        thumb=raw.thumb,
        extra=raw.extra,
        xml=raw.xml,
        raw=raw,
    )
    if msg.type == MsgType.FRIEND_CONFIRM:
        try:
            from xml.etree import ElementTree as ET

            root = ET.fromstring(raw.content)
            if root.tag == "msg":
                msg.new_friend_request = NewFriendRequest(
                    v3=root.attrib.get("encryptusername", ""),
                    v4=root.attrib.get("ticket", ""),
                    scene=int(root.attrib.get("scene", "0") or "0"),
                )
        except ET.ParseError:  # pragma: no cover
            msg.new_friend_request = None
    return msg


__all__ = [
    "Contact",
    "ContactFlags",
    "GenderType",
    "Message",
    "MessageBuffer",
    "MsgType",
    "NewFriendRequest",
    "SelfInfo",
    "contact_from_proto",
    "message_from_proto",
]
