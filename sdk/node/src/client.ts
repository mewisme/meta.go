import {Metadata, createChannel, createClientFactory, type Channel} from "nice-grpc"
import {errorMiddleware, MetaError, ProtocolMismatchError, UnsupportedCapabilityError} from "./errors.js"
import {EventStream} from "./events.js"
import {E2EEServiceDefinition, type E2EEServiceClient} from "./gen/meta/v1/e2ee.js"
import {FacebookServiceDefinition, type FacebookServiceClient} from "./gen/meta/v1/facebook.js"
import {MessengerServiceDefinition, type MessengerServiceClient} from "./gen/meta/v1/messenger.js"
import {RuntimeServiceDefinition, type GetInfoResponse, type RuntimeServiceClient} from "./gen/meta/v1/runtime.js"
import {SessionServiceDefinition, type AuthSnapshot as ProtoAuthSnapshot, type ConnectResponse, type CreateSessionResponse, type GetHealthResponse, type SessionAuth as ProtoSessionAuth, type SessionServiceClient} from "./gen/meta/v1/session.js"
import {ManagedRuntime, PROTOCOL_MAJOR, type ManagedRuntimeOptions} from "./runtime.js"

export type RuntimeOptions = {mode: "external"; endpoint: string; token: string} | ({mode: "managed"} & ManagedRuntimeOptions)
export type Credentials = {identifier: string; password: string} & ({totp: string; otp?: never} | {otp: string; totp?: never})
export interface AppStateCookie { key: string; value: string; domain?: string; path?: string; hostOnly?: boolean; secure?: boolean; httpOnly?: boolean }
export type SessionAuth =
  | {cookies: Record<string, string>; appState?: never; credentials?: never}
  | {appState: AppStateCookie[]; cookies?: never; credentials?: never}
  | {credentials: Credentials; cookies?: never; appState?: never}
export interface FacebookSession { accountId: string; name: string; username: string; dtsg: string; jazoest: string; lsd: string; sessionId: string; clientRevision: bigint; refreshedAt?: Date }
export interface AuthSnapshot { cookies: Record<string, string>; appState: AppStateCookie[]; session?: FacebookSession }
export interface CreateSessionOptions { e2ee?: boolean; eventBuffer?: number; timeoutMs?: number }
export interface EventOptions { buffer?: number; iteratorBuffer?: number }

export class MetaClient {
  readonly endpoint: string
  readonly runtime: RuntimeServiceClient
  readonly sessions: SessionServiceClient
  readonly messenger: MessengerServiceClient
  readonly e2ee: E2EEServiceClient
  readonly facebook: FacebookServiceClient
  private readonly channel: Channel
  private readonly managed?: ManagedRuntime
  private capabilities = new Set<string>()
  private readonly eventStreams = new Set<EventStream>()
  private runtimeInfo?: GetInfoResponse
  private closed = false

  private constructor(endpoint: string, token: string, managed?: ManagedRuntime) {
    this.endpoint = endpoint
    this.managed = managed
    this.channel = createChannel(endpoint)
    const metadata = Metadata({authorization: `Bearer ${token}`})
    const factory = createClientFactory().use(errorMiddleware)
    const defaults = {"*": {metadata}}
    this.runtime = factory.create(RuntimeServiceDefinition, this.channel, defaults)
    this.sessions = factory.create(SessionServiceDefinition, this.channel, defaults)
    this.messenger = factory.create(MessengerServiceDefinition, this.channel, defaults)
    this.e2ee = factory.create(E2EEServiceDefinition, this.channel, defaults)
    this.facebook = factory.create(FacebookServiceDefinition, this.channel, defaults)
  }

  static async create(options: RuntimeOptions): Promise<MetaClient> {
    const managed = options.mode === "managed" ? await ManagedRuntime.start(options) : undefined
    const endpoint = managed?.endpoint ?? (options.mode === "external" ? options.endpoint : "")
    const token = managed?.token ?? (options.mode === "external" ? options.token : "")
    const client = new MetaClient(endpoint, token, managed)
    try {
      const info = await client.runtime.getInfo({})
      const major = info.protocol?.major ?? 0
      if (major !== PROTOCOL_MAJOR) throw new ProtocolMismatchError(PROTOCOL_MAJOR, major)
      client.runtimeInfo = info
      client.capabilities = new Set(info.capabilities)
      return client
    } catch (error) {
      await client.close()
      throw error
    }
  }

  info(): GetInfoResponse | undefined { return this.runtimeInfo }
  hasCapability(capability: string): boolean { return this.capabilities.has(capability) }
  requireCapability(capability: string): void {
    if (!this.hasCapability(capability)) throw new UnsupportedCapabilityError(capability)
  }

  createSession(auth: SessionAuth, options: CreateSessionOptions = {}): Promise<CreateSessionResponse> {
    this.requireAuthCapability(auth)
    return this.sessions.createSession({auth: sessionAuthToProto(auth), e2ee: options.e2ee ?? false, eventBuffer: options.eventBuffer ?? 0, timeout: options.timeoutMs === undefined ? undefined : durationFromMilliseconds(options.timeoutMs), cookies: {}})
  }

  async refreshAuth(sessionId: string, auth?: SessionAuth): Promise<AuthSnapshot> {
    this.requireCapability("session.auth.refresh")
    if (auth) this.requireAuthCapability(auth)
    const response = await this.sessions.refreshAuth({sessionId, auth: auth ? sessionAuthToProto(auth) : undefined})
    return authSnapshotFromProto(response.snapshot)
  }

  async authSnapshot(sessionId: string): Promise<AuthSnapshot> {
    this.requireCapability("session.auth.refresh")
    const response = await this.sessions.getAuthSnapshot({sessionId})
    return authSnapshotFromProto(response.snapshot)
  }

  private requireAuthCapability(auth: SessionAuth): void {
    if (auth.appState !== undefined) this.requireCapability("session.auth.app_state")
    if (auth.credentials !== undefined) this.requireCapability("session.auth.credentials")
  }

  connectSession(sessionId: string): Promise<ConnectResponse> { return this.sessions.connect({sessionId}) }
  getSessionHealth(sessionId: string): Promise<GetHealthResponse> { return this.sessions.getHealth({sessionId}) }
  async closeSession(sessionId: string): Promise<void> { await this.sessions.closeSession({sessionId}) }

  events(sessionId: string, options: EventOptions = {}): EventStream {
    const abort = new AbortController()
    const source = this.sessions.subscribeEvents({sessionId, buffer: options.buffer ?? 0}, {signal: abort.signal})
    const stream = new EventStream(source, abort, options.iteratorBuffer ?? 256, () => this.eventStreams.delete(stream))
    this.eventStreams.add(stream)
    return stream
  }

  async close(): Promise<void> {
    if (this.closed) return
    this.closed = true
    for (const stream of this.eventStreams) stream.close()
    this.eventStreams.clear()
    this.channel.close()
    await this.managed?.stop()
  }
}

function sessionAuthToProto(auth: SessionAuth): ProtoSessionAuth {
  if (auth.cookies !== undefined) return {source: {$case: "cookies", value: {values: {...auth.cookies}}}}
  if (auth.appState !== undefined) {
    if (auth.appState.length === 0) throw new TypeError("appState must contain at least one cookie")
    const cookies = auth.appState.map(cookie => {
      if (cookie.key.trim() === "" || cookie.value === "") throw new TypeError("appState cookies require key and value")
      return {key: cookie.key, value: cookie.value, domain: cookie.domain ?? "", path: cookie.path ?? "", hostOnly: cookie.hostOnly ?? false, secure: cookie.secure ?? false, httpOnly: cookie.httpOnly ?? false}
    })
    return {source: {$case: "appState", value: {cookies}}}
  }
  const credentials = auth.credentials
  if (!credentials) throw new TypeError("exactly one authentication source is required")
  if (credentials.identifier.trim() === "") throw new TypeError("credentials.identifier is required")
  if (credentials.password === "") throw new TypeError("credentials.password is required")
  const hasTotp = typeof credentials.totp === "string" && credentials.totp.trim() !== ""
  const hasOtp = typeof credentials.otp === "string" && credentials.otp.trim() !== ""
  if (hasTotp === hasOtp) throw new TypeError("credentials require exactly one of totp or otp")
  return {source: {$case: "credentials", value: {identifier: credentials.identifier, password: credentials.password, secondFactor: hasTotp ? {$case: "totp", value: credentials.totp!} : {$case: "otp", value: credentials.otp!}}}}
}

function authSnapshotFromProto(snapshot: ProtoAuthSnapshot | undefined): AuthSnapshot {
  if (!snapshot) throw new MetaError("runtime returned no authentication snapshot")
  const session = snapshot.session
  return {
    cookies: {...(snapshot.cookies?.values ?? {})},
    appState: (snapshot.appState?.cookies ?? []).map(cookie => ({key: cookie.key, value: cookie.value, domain: cookie.domain || undefined, path: cookie.path || undefined, hostOnly: cookie.hostOnly, secure: cookie.secure, httpOnly: cookie.httpOnly})),
    session: session ? {accountId: session.accountId, name: session.name, username: session.username, dtsg: session.dtsg, jazoest: session.jazoest, lsd: session.lsd, sessionId: session.sessionId, clientRevision: session.clientRevision, refreshedAt: session.refreshedAt} : undefined,
  }
}

function durationFromMilliseconds(value: number): {seconds: bigint; nanos: number} {
  if (!Number.isFinite(value) || value <= 0) throw new RangeError("timeoutMs must be positive")
  const milliseconds = Math.trunc(value)
  return {seconds: BigInt(Math.trunc(milliseconds / 1000)), nanos: (milliseconds % 1000) * 1_000_000}
}
