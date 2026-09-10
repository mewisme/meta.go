from __future__ import annotations

import asyncio

import pytest

from meewmeew_meta import EventBufferOverflowError, EventStream
from meta.v1 import events_pb2, session_pb2


def ready(sequence: int = 1) -> events_pb2.Event:
    return events_pb2.Event(session_id="s1", sequence=sequence, ready=events_pb2.ReadyEvent(is_new_session=True))


def typing(sequence: int = 2) -> events_pb2.Event:
    return events_pb2.Event(
        session_id="s1",
        sequence=sequence,
        typing=events_pb2.TypingEvent(thread_id="t1", sender_id="u1", typing=True),
    )


@pytest.mark.asyncio
async def test_event_stream_broadcasts_one_source_to_iterator_and_listener() -> None:
    gate = asyncio.Event()

    async def source():
        await gate.wait()
        yield session_pb2.SubscribeEventsResponse(event=ready())
        yield session_pb2.SubscribeEventsResponse(event=typing())

    stream = EventStream(source())
    iterator = stream.__aiter__()
    first = asyncio.create_task(anext(iterator))
    seen: list[events_pb2.Event] = []
    unsubscribe = stream.on("ready", seen.append)
    gate.set()

    assert await first == ready()
    assert await anext(iterator) == typing()
    with pytest.raises(StopAsyncIteration):
        await anext(iterator)
    assert seen == [ready()]
    unsubscribe()


@pytest.mark.asyncio
async def test_event_stream_drains_buffer_on_normal_completion() -> None:
    gate = asyncio.Event()

    async def source():
        await gate.wait()
        yield session_pb2.SubscribeEventsResponse(event=ready())
        yield session_pb2.SubscribeEventsResponse(event=typing())

    stream = EventStream(source(), iterator_buffer=2)
    iterator = stream.__aiter__()
    first = asyncio.create_task(anext(iterator))
    gate.set()

    assert await first == ready()
    await asyncio.sleep(0)
    assert await anext(iterator) == typing()
    with pytest.raises(StopAsyncIteration):
        await anext(iterator)


@pytest.mark.asyncio
async def test_event_stream_fails_only_slow_iterator_on_overflow() -> None:
    gate = asyncio.Event()

    async def source():
        await gate.wait()
        for sequence in range(1, 5):
            yield session_pb2.SubscribeEventsResponse(event=ready(sequence))
            await asyncio.sleep(0)

    stream = EventStream(source(), iterator_buffer=1)
    slow = stream.__aiter__()
    fast = stream.__aiter__()
    slow_first = asyncio.create_task(anext(slow))
    fast_first = asyncio.create_task(anext(fast))
    gate.set()

    assert (await slow_first).sequence == 1
    assert (await fast_first).sequence == 1
    assert (await anext(fast)).sequence == 2
    await asyncio.sleep(0)
    with pytest.raises(EventBufferOverflowError):
        await anext(slow)
    assert (await anext(fast)).sequence == 3
    assert (await anext(fast)).sequence == 4


@pytest.mark.asyncio
async def test_event_stream_close_cancels_source_once() -> None:
    cancelled = 0

    async def source():
        await asyncio.Future()
        yield session_pb2.SubscribeEventsResponse()

    def cancel() -> None:
        nonlocal cancelled
        cancelled += 1

    stream = EventStream(source(), cancel=cancel)
    stream.on("ready", lambda _: None)
    await asyncio.sleep(0)
    await stream.aclose()
    await stream.aclose()
    assert cancelled == 1
