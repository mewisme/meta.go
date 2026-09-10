import assert from "node:assert/strict"
import test from "node:test"
import {EventBufferOverflowError, EventStream, type events} from "../src/index.js"

const ready: events.Event = {sessionId: "s1", sequence: 1n, emittedAt: undefined, payload: {$case: "ready", value: {isNewSession: true}}}
const typing: events.Event = {sessionId: "s1", sequence: 2n, emittedAt: undefined, payload: {$case: "typing", value: {threadId: "t1", senderId: "u1", typing: true}}}

test("EventStream broadcasts one source to iterator and typed listeners", async () => {
  async function* source() {
    await Promise.resolve()
    yield {event: ready}
    yield {event: typing}
  }
  const stream = new EventStream(source(), new AbortController())
  const iterator = stream[Symbol.asyncIterator]()
  const first = iterator.next()
  let listenerEvent: events.Event | undefined
  const off = stream.on("ready", event => {
    assert.equal(event.payload.value.isNewSession, true)
    listenerEvent = event
  })
  assert.deepEqual((await first).value, ready)
  assert.deepEqual((await iterator.next()).value, typing)
  assert.equal((await iterator.next()).done, true)
  assert.deepEqual(listenerEvent, ready)
  off()
})

test("EventStream fails only a slow iterator when its local queue overflows", async () => {
  let release!: () => void
  const gate = new Promise<void>(resolve => { release = resolve })
  async function* source() {
    await gate
    yield {event: ready}
    yield {event: typing}
    yield {event: ready}
  }
  const stream = new EventStream(source(), new AbortController(), 1)
  const iterator = stream[Symbol.asyncIterator]()
  const first = iterator.next()
  release()
  assert.deepEqual((await first).value, ready)
  await new Promise(resolve => setTimeout(resolve, 10))
  assert.deepEqual((await iterator.next()).value, typing)
  await assert.rejects(iterator.next(), EventBufferOverflowError)
  stream.close()
})
