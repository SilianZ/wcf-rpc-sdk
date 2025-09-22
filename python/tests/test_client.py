from __future__ import annotations

import threading

import pytest

from wcf_rpc_sdk import Client, MsgType
from wcf_rpc_sdk import wcf_pb2


class DummyTransport:
    def __init__(self, messages: list[wcf_pb2.WxMsg] | None = None) -> None:
        self.messages = messages or []
        self.sent_text: list[tuple[str, str, tuple[str, ...]]] = []
        self.sent_images: list[tuple[str, str]] = []
        self.sent_files: list[tuple[str, str]] = []
        self.accepted_friends: list[tuple[str, str, int]] = []
        self._closed = False
        self._enabled = False

    def enable_recv_text(self) -> int:
        self._enabled = True
        return 1

    def disable_recv_text(self) -> int:
        self._enabled = False
        return 1

    def listen(self, callback, stop_event: threading.Event) -> None:
        for msg in self.messages:
            callback(msg)
        stop_event.wait(0.05)

    def close(self) -> None:
        self._closed = True

    def send_text(self, receiver: str, content: str, ats: tuple[str, ...]):
        self.sent_text.append((receiver, content, tuple(ats)))
        return 0

    def send_image(self, receiver: str, path: str) -> int:
        self.sent_images.append((receiver, path))
        return 0

    def send_file(self, receiver: str, path: str) -> int:
        self.sent_files.append((receiver, path))
        return 0

    def send_xml(self, receiver: str, content: str, path: str = "", msg_type: int = 0) -> int:
        return 0

    def send_rich_text(self, receiver: str, **kwargs: str) -> int:
        return 1

    def accept_friend(self, v3: str, v4: str, scene: int) -> int:
        self.accepted_friends.append((v3, v4, scene))
        return 1

    def is_login(self) -> bool:
        return True

    def get_user_info(self):
        return wcf_pb2.UserInfo(wxid="wxid_dummy", name="Robot", mobile="", home="C:/WeChat")

    def get_contacts(self):
        return [
            wcf_pb2.RpcContact(wxid="wxid_friend", name="Friend"),
            wcf_pb2.RpcContact(wxid="test@chatroom", name="Room"),
        ]


@pytest.fixture()
def text_message() -> wcf_pb2.WxMsg:
    return wcf_pb2.WxMsg(
        is_self=False,
        is_group=False,
        id=123,
        type=MsgType.TEXT.value,
        ts=1,
        roomid="",
        content="hello",
        sender="wxid_friend",
    )


def test_client_receives_messages(text_message):
    dummy = DummyTransport(messages=[text_message])
    client = Client(msg_buffer_size=2, transport_factory=lambda addr: dummy)
    client.run()
    msg = client.get_message(timeout=1)
    assert msg.content == "hello"
    assert msg.message_id == 123
    client.close()


def test_message_reply_helpers(text_message):
    dummy = DummyTransport(messages=[text_message])
    client = Client(msg_buffer_size=2, transport_factory=lambda addr: dummy)
    client.run()
    msg = client.get_message(timeout=1)
    msg.reply_text("world")
    msg.reply_image("/tmp/image.jpg")
    msg.reply_file("/tmp/file.txt")
    assert dummy.sent_text == [("wxid_friend", "world", tuple())]
    assert dummy.sent_images == [("wxid_friend", "/tmp/image.jpg")]
    assert dummy.sent_files == [("wxid_friend", "/tmp/file.txt")]
    client.close()


def test_accept_new_friend_parses_xml():
    xml = "<msg encryptusername=\"v3\" ticket=\"v4\" scene=\"1\" />"
    wxmsg = wcf_pb2.WxMsg(
        is_self=False,
        is_group=False,
        id=1,
        type=MsgType.FRIEND_CONFIRM.value,
        ts=1,
        roomid="",
        content=xml,
        sender="wxid_stranger",
    )
    dummy = DummyTransport(messages=[wxmsg])
    client = Client(msg_buffer_size=2, transport_factory=lambda addr: dummy)
    client.run()
    msg = client.get_message(timeout=1)
    assert msg.new_friend_request is not None
    assert client.accept_new_friend(msg)
    assert dummy.accepted_friends == [("v3", "v4", 1)]
    client.close()


def test_contact_cache():
    dummy = DummyTransport()
    client = Client(msg_buffer_size=1, transport_factory=lambda addr: dummy)
    contacts = client.get_contacts()
    assert {c.wxid for c in contacts} == {"wxid_friend", "test@chatroom"}
    friends = client.get_all_friend()
    assert friends and friends[0].wxid == "wxid_friend"
    rooms = client.get_all_chat_room()
    assert rooms and rooms[0].wxid == "test@chatroom"
    info = client.get_self_info()
    assert info and info.wxid == "wxid_dummy"
    client.close()
