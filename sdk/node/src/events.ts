import {EventBufferOverflowError} from "./errors.js"
import type {Event} from "./gen/meta/v1/events.js"
import type {SubscribeEventsResponse} from "./gen/meta/v1/session.js"

type EventPayload = NonNullable<Event["payload"]>
export type EventName = EventPayload["$case"]
export type EventOf<Name extends EventName> = Event & {payload: Extract<EventPayload, {$case: Name}>}
export type EventListener<Name extends EventName> = (event: EventOf<Name>) => void

class EventQueue {
  private values: Event[] = []
  private waiters: Array<{resolve: (value: IteratorResult<Event>) => void; reject: (error: unknown) => void}> = []
  private error: unknown
  private ended = false
  constructor(private readonly limit: number) {}

  push(value: Event): void {
    if (this.ended) return
    const waiter = this.waiters.shift()
    if (waiter) return waiter.resolve({value, done: false})
    if (this.values.length >= this.limit) return this.fail(new EventBufferOverflowError(this.limit))
    this.values.push(value)
  }

  next(): Promise<IteratorResult<Event>> {
    if (this.values.length) return Promise.resolve({value: this.values.shift()!, done: false})
    if (this.error) return Promise.reject(this.error)
    if (this.ended) return Promise.resolve({value: undefined, done: true})
    return new Promise((resolve, reject) => this.waiters.push({resolve, reject}))
  }

  close(): void {
    if (this.ended) return
    this.ended = true
    for (const waiter of this.waiters.splice(0)) waiter.resolve({value: undefined, done: true})
  }

  fail(error: unknown): void {
    if (this.ended) return
    this.error = error
    this.ended = true
    for (const waiter of this.waiters.splice(0)) waiter.reject(error)
  }
}

export class EventStream implements AsyncIterable<Event> {
  private readonly listeners = new Map<EventName, Set<(event: Event) => void>>()
  private readonly queues = new Set<EventQueue>()
  private started = false
  private closed = false

  constructor(private readonly source: AsyncIterable<SubscribeEventsResponse>, private readonly abort: AbortController, private readonly iteratorBuffer = 256, private readonly onClose?: () => void) {}

  on<Name extends EventName>(name: Name, listener: EventListener<Name>): () => void {
    let listeners = this.listeners.get(name)
    if (!listeners) this.listeners.set(name, listeners = new Set())
    const value = listener as (event: Event) => void
    listeners.add(value)
    this.start()
    return () => listeners?.delete(value)
  }

  close(): void {
    if (this.closed) return
    this.closed = true
    this.abort.abort()
    for (const queue of this.queues) queue.close()
    this.queues.clear()
    this.listeners.clear()
    this.onClose?.()
  }

  async *[Symbol.asyncIterator](): AsyncIterator<Event> {
    if (this.closed) return
    const queue = new EventQueue(this.iteratorBuffer)
    this.queues.add(queue)
    this.start()
    try {
      while (true) {
        const next = await queue.next()
        if (next.done) return
        yield next.value
      }
    } finally {
      this.queues.delete(queue)
    }
  }

  private start(): void {
    if (this.started || this.closed) return
    this.started = true
    void this.pump()
  }

  private async pump(): Promise<void> {
    try {
      for await (const response of this.source) {
        if (this.closed) return
        const event = response.event
        if (!event?.payload) continue
        for (const listener of this.listeners.get(event.payload.$case) ?? []) listener(event)
        for (const queue of this.queues) queue.push(event)
      }
      this.closed = true
      for (const queue of this.queues) queue.close()
      this.listeners.clear()
      this.onClose?.()
    } catch (error) {
      if (this.closed) return
      this.closed = true
      for (const queue of this.queues) queue.fail(error)
      this.listeners.clear()
      this.onClose?.()
    }
  }
}
