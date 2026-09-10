from __future__ import annotations

import asyncio
import contextlib
import inspect
from collections import defaultdict
from collections.abc import AsyncIterator, Awaitable, Callable
from typing import Literal, TypeAlias

from meta.v1 import events_pb2, session_pb2

from .errors import EventBufferOverflowError, map_rpc_error

EventName: TypeAlias = Literal[
    "ready",
    "reconnected",
    "disconnected",
    "error",
    "message",
    "reaction",
    "typing",
    "read_receipt",
    "delivery_receipt",
    "message_edit",
    "message_unsend",
    "thread_update",
    "e2ee_ready",
    "e2ee_receipt",
    "thread_system",
]
EventListener: TypeAlias = Callable[[events_pb2.Event], None | Awaitable[None]]
_END = object()


class _IteratorState:
    def __init__(self, limit: int) -> None:
        self.limit = limit
        self.queue: asyncio.Queue[events_pb2.Event | object] = asyncio.Queue(maxsize=limit + 1)
        self.error: BaseException | None = None
        self.closed = False

    def push(self, event: events_pb2.Event) -> None:
        if self.closed:
            return
        if self.queue.qsize() >= self.limit:
            self.fail(EventBufferOverflowError(self.limit))
            return
        self.queue.put_nowait(event)

    def close(self) -> None:
        if self.closed:
            return
        self.closed = True
        self.queue.put_nowait(_END)

    def fail(self, error: BaseException) -> None:
        if self.closed:
            return
        self.error = error
        self.closed = True
        self._clear()
        self.queue.put_nowait(_END)

    def _clear(self) -> None:
        while not self.queue.empty():
            self.queue.get_nowait()


class EventStream:
    def __init__(
        self,
        source: AsyncIterator[session_pb2.SubscribeEventsResponse],
        cancel: Callable[[], object] | None = None,
        iterator_buffer: int = 256,
        on_close: Callable[[EventStream], None] | None = None,
    ) -> None:
        if iterator_buffer < 1:
            raise ValueError("iterator_buffer must be positive")
        self._source = source
        self._cancel = cancel
        self._iterator_buffer = iterator_buffer
        self._on_close = on_close
        self._listeners: dict[str, set[EventListener]] = defaultdict(set)
        self._iterators: set[_IteratorState] = set()
        self._listener_tasks: set[asyncio.Future[None]] = set()
        self._pump_task: asyncio.Task[None] | None = None
        self._closed = False

    def on(self, name: EventName, listener: EventListener) -> Callable[[], None]:
        self._listeners[name].add(listener)
        self._start()

        def unsubscribe() -> None:
            self._listeners[name].discard(listener)

        return unsubscribe

    def __aiter__(self) -> AsyncIterator[events_pb2.Event]:
        return self._iterate()

    async def aclose(self) -> None:
        if self._closed:
            return
        self._closed = True
        if self._cancel:
            self._cancel()
        if self._pump_task and self._pump_task is not asyncio.current_task():
            self._pump_task.cancel()
            with contextlib.suppress(asyncio.CancelledError):
                await self._pump_task
        self._finish()

    async def _iterate(self) -> AsyncIterator[events_pb2.Event]:
        if self._closed:
            return
        state = _IteratorState(self._iterator_buffer)
        self._iterators.add(state)
        self._start()
        try:
            while True:
                item = await state.queue.get()
                if item is _END:
                    if state.error:
                        raise state.error
                    return
                assert isinstance(item, events_pb2.Event)
                yield item
        finally:
            self._iterators.discard(state)

    def _start(self) -> None:
        if self._pump_task is None and not self._closed:
            self._pump_task = asyncio.create_task(self._pump())

    async def _pump(self) -> None:
        try:
            async for response in self._source:
                if self._closed:
                    return
                if not response.HasField("event"):
                    continue
                event = response.event
                name = event.WhichOneof("payload")
                if name:
                    for listener in tuple(self._listeners.get(name, ())):
                        self._dispatch_listener(listener, event)
                for state in tuple(self._iterators):
                    state.push(event)
        except asyncio.CancelledError:
            raise
        except BaseException as error:
            mapped = map_rpc_error(error)
            for state in tuple(self._iterators):
                state.fail(mapped)
        finally:
            self._closed = True
            self._finish()

    def _dispatch_listener(self, listener: EventListener, event: events_pb2.Event) -> None:
        result = listener(event)
        if not inspect.isawaitable(result):
            return
        task = asyncio.ensure_future(result)
        self._listener_tasks.add(task)
        task.add_done_callback(self._listener_tasks.discard)

    def _finish(self) -> None:
        for state in tuple(self._iterators):
            state.close()
        for task in tuple(self._listener_tasks):
            task.cancel()
        self._listeners.clear()
        if self._on_close:
            callback, self._on_close = self._on_close, None
            callback(self)
