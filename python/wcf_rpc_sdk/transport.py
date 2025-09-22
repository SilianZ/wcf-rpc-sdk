"""Low level transport layer for the Python SDK."""
from __future__ import annotations

import threading
from typing import Callable, Iterable, List, Optional

from .exceptions import DependencyMissingError, TransportClosedError
from . import wcf_pb2
from .utils import increment_port, join_aters, normalize_address

try:  # pragma: no cover
    import pynng
except ModuleNotFoundError:  # pragma: no cover
    pynng = None


MessageCallback = Callable[[wcf_pb2.WxMsg], None]


class NanomsgTransport:
    def __init__(self, address: Optional[str] = None) -> None:
        if pynng is None:  # pragma: no cover
            raise DependencyMissingError(
                "The pynng package is required to talk to the WCF RPC service. "
                "Install it with `pip install pynng`."
            )
        self.address = normalize_address(address)
        self._socket = pynng.Pair1()
        self._socket.dial(self.address, block=True)
        self._socket.send_timeout = 5_000
        self._socket.recv_timeout = 5_000
        self._lock = threading.RLock()
        self._closed = False
        self._recv_enabled = False

    def _ensure_open(self) -> None:
        if self._closed:
            raise TransportClosedError("transport has been closed")

    def _request(self, request: wcf_pb2.Request) -> wcf_pb2.Response:
        self._ensure_open()
        with self._lock:
            self._socket.send(request.SerializeToString())
            raw = self._socket.recv()
        response = wcf_pb2.Response()
        response.ParseFromString(raw)
        return response

    def _call(self, func: wcf_pb2.Functions) -> wcf_pb2.Response:
        return self._request(wcf_pb2.Request(func=func))

    def is_login(self) -> bool:
        return self._call(wcf_pb2.FUNC_IS_LOGIN).status == 1

    def get_self_wxid(self) -> str:
        return self._call(wcf_pb2.FUNC_GET_SELF_WXID).str

    def get_msg_types(self) -> dict[int, str]:
        return dict(self._call(wcf_pb2.FUNC_GET_MSG_TYPES).types.types)

    def get_contacts(self) -> List[wcf_pb2.RpcContact]:
        return list(self._call(wcf_pb2.FUNC_GET_CONTACTS).contacts.contacts)

    def get_user_info(self) -> Optional[wcf_pb2.UserInfo]:
        response = self._call(wcf_pb2.FUNC_GET_USER_INFO)
        if response.HasField("ui"):
            return response.ui
        return None

    def get_db_names(self) -> List[str]:
        return list(self._call(wcf_pb2.FUNC_GET_DB_NAMES).dbs.names)

    def get_db_tables(self, name: str) -> List[wcf_pb2.DbTable]:
        request = wcf_pb2.Request(func=wcf_pb2.FUNC_GET_DB_TABLES, str=name)
        return list(self._request(request).tables.tables)

    def exec_db_query(self, database: str, sql: str) -> List[wcf_pb2.DbRow]:
        request = wcf_pb2.Request(
            func=wcf_pb2.FUNC_EXEC_DB_QUERY,
            query=wcf_pb2.DbQuery(db=database, sql=sql),
        )
        return list(self._request(request).rows.rows)

    def send_text(self, receiver: str, content: str, ats: Iterable[str] = ()) -> int:
        request = wcf_pb2.Request(
            func=wcf_pb2.FUNC_SEND_TXT,
            txt=wcf_pb2.TextMsg(msg=content, receiver=receiver, aters=join_aters(ats)),
        )
        return self._request(request).status

    def send_image(self, receiver: str, path: str) -> int:
        request = wcf_pb2.Request(
            func=wcf_pb2.FUNC_SEND_IMG,
            file=wcf_pb2.PathMsg(path=path, receiver=receiver),
        )
        return self._request(request).status

    def send_file(self, receiver: str, path: str) -> int:
        request = wcf_pb2.Request(
            func=wcf_pb2.FUNC_SEND_FILE,
            file=wcf_pb2.PathMsg(path=path, receiver=receiver),
        )
        return self._request(request).status

    def send_rich_text(self, receiver: str, **kwargs: str) -> int:
        request = wcf_pb2.Request(
            func=wcf_pb2.FUNC_SEND_RICH_TXT,
            rt=wcf_pb2.RichText(receiver=receiver, **kwargs),
        )
        return self._request(request).status

    def send_xml(self, receiver: str, content: str, path: str = "", msg_type: int = 0) -> int:
        request = wcf_pb2.Request(
            func=wcf_pb2.FUNC_SEND_XML,
            xml=wcf_pb2.XmlMsg(receiver=receiver, content=content, path=path, type=msg_type),
        )
        return self._request(request).status

    def accept_friend(self, v3: str, v4: str, scene: int) -> int:
        request = wcf_pb2.Request(
            func=wcf_pb2.FUNC_ACCEPT_FRIEND,
            v=wcf_pb2.Verification(v3=v3, v4=v4, scene=scene),
        )
        return self._request(request).status

    def enable_recv_text(self) -> int:
        request = wcf_pb2.Request(func=wcf_pb2.FUNC_ENABLE_RECV_TXT, flag=True)
        status = self._request(request).status
        if status == 1:
            self._recv_enabled = True
        return status

    def disable_recv_text(self) -> int:
        status = self._call(wcf_pb2.FUNC_DISABLE_RECV_TXT).status
        self._recv_enabled = False
        return status

    def listen(self, callback: MessageCallback, stop_event: threading.Event) -> None:
        if pynng is None:  # pragma: no cover
            raise DependencyMissingError("The pynng package is required to listen for messages.")
        if not self._recv_enabled:
            return
        with pynng.Pair1() as sock:
            sock.dial(increment_port(self.address), block=True)
            sock.recv_timeout = 5_000
            sock.send_timeout = 5_000
            while not stop_event.is_set() and self._recv_enabled:
                try:
                    raw = sock.recv()
                except pynng.Timeout:  # pragma: no cover
                    continue
                response = wcf_pb2.Response()
                response.ParseFromString(raw)
                if response.HasField("wxmsg"):
                    callback(response.wxmsg)

    def close(self) -> None:
        if self._closed:
            return
        self._socket.close()
        self._closed = True
        self._recv_enabled = False


__all__ = ["NanomsgTransport", "MessageCallback"]
