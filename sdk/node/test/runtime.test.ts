import assert from "node:assert/strict"
import {execFileSync} from "node:child_process"
import {mkdtempSync, rmSync} from "node:fs"
import {tmpdir} from "node:os"
import {resolve} from "node:path"
import {fileURLToPath} from "node:url"
import test, {after, before} from "node:test"
import {ManagedRuntime, MetaClient, MetaRpcError, UnsupportedCapabilityError, type SessionAuth} from "../src/index.js"

const sdkDir = fileURLToPath(new URL("../", import.meta.url))
const repoRoot = resolve(sdkDir, "../..")
const buildDir = mkdtempSync(resolve(tmpdir(), "meta-node-test-"))
const runtimePath = resolve(buildDir, process.platform === "win32" ? "meta-runtime.exe" : "meta-runtime")

before(() => execFileSync("go", ["build", "-o", runtimePath, "./cmd/meta-runtime"], {cwd: repoRoot, env: {...process.env, GOWORK: "off"}, stdio: "inherit"}))
after(() => rmSync(buildDir, {recursive: true, force: true}))

test("managed runtime exposes capabilities and session lifecycle", async () => {
  const client = await MetaClient.create({mode: "managed", runtimePath})
  try {
    assert.equal(client.info()?.protocol?.major, 1)
    assert.equal(client.hasCapability("session.lifecycle"), true)
    assert.equal(client.hasCapability("session.auth.app_state"), true)
    assert.equal(client.hasCapability("session.auth.credentials"), true)
    assert.equal(client.hasCapability("session.auth.refresh"), true)
    assert.throws(() => client.requireCapability("future.echo"), error => error instanceof UnsupportedCapabilityError && error.capability === "future.echo")
    const created = await client.createSession({cookies: {c_user: "42", xs: "test"}}, {eventBuffer: 8, timeoutMs: 5000})
    assert.ok(created.sessionId)
    const appState = await client.createSession({appState: [{key: "c_user", value: "42"}, {key: "xs", value: "test", httpOnly: true}]})
    assert.ok(appState.sessionId)
    const credentials = await client.createSession({credentials: {identifier: "user", password: "password", otp: "123456"}})
    assert.ok(credentials.sessionId)
    assert.throws(() => client.createSession({credentials: {identifier: "user", password: "password"}} as unknown as SessionAuth), /exactly one of totp or otp/)
    assert.throws(() => client.createSession({credentials: {identifier: "user", password: "password", totp: "totp-secret", otp: "123456"}} as unknown as SessionAuth), /exactly one of totp or otp/)
    const health = await client.getSessionHealth(created.sessionId)
    assert.ok(health.health)
    await client.closeSession(created.sessionId)
    await client.closeSession(appState.sessionId)
    await client.closeSession(credentials.sessionId)
  } finally {
    await client.close()
  }
})

test("external endpoint mode authenticates against an existing runtime", async () => {
  const runtime = await ManagedRuntime.start({runtimePath})
  try {
    const client = await MetaClient.create({mode: "external", endpoint: runtime.endpoint, token: runtime.token})
    try {
      assert.equal(client.info()?.protocol?.major, 1)
    } finally {
      await client.close()
    }
    await assert.rejects(MetaClient.create({mode: "external", endpoint: runtime.endpoint, token: "wrong-token"}), MetaRpcError)
  } finally {
    await runtime.stop()
  }
})

test("auth wrappers preserve client event state and enforce runtime capabilities", async () => {
  const client = await MetaClient.create({mode: "managed", runtimePath})
  const internal = client as unknown as {capabilities: Set<string>; eventStreams: Set<{close(): void}>}
  const originalCapabilities = internal.capabilities
  try {
    internal.capabilities = new Set(["session.lifecycle"])
    assert.throws(() => client.createSession({appState: [{key: "c_user", value: "42"}, {key: "xs", value: "x"}]}), UnsupportedCapabilityError)
    assert.throws(() => client.createSession({credentials: {identifier: "user", password: "x", otp: "123456"}}), UnsupportedCapabilityError)
    await assert.rejects(client.refreshAuth("session-1"), UnsupportedCapabilityError)
    await assert.rejects(client.authSnapshot("session-1"), UnsupportedCapabilityError)
    internal.capabilities = originalCapabilities

    const snapshot = {
      cookies: {values: {c_user: "42", xs: "fresh"}},
      appState: {cookies: [{key: "c_user", value: "42", domain: "", path: "", hostOnly: false, secure: false, httpOnly: false}]},
      session: {accountId: "42", name: "Mew", username: "mew", dtsg: "d", jazoest: "j", lsd: "l", sessionId: "facebook-session", clientRevision: 7n, refreshedAt: new Date(0)},
    }
    let refreshSessionId = ""
    let snapshotSessionId = ""
    Object.defineProperty(client, "sessions", {value: {
      refreshAuth: async (request: {sessionId: string}) => { refreshSessionId = request.sessionId; return {snapshot} },
      getAuthSnapshot: async (request: {sessionId: string}) => { snapshotSessionId = request.sessionId; return {snapshot} },
    }})
    let closed = false
    const marker = {close: () => { closed = true }}
    internal.eventStreams.add(marker)
    const refreshed = await client.refreshAuth("session-1")
    const current = await client.authSnapshot("session-1")
    assert.equal(refreshSessionId, "session-1")
    assert.equal(snapshotSessionId, "session-1")
    assert.equal(refreshed.cookies.xs, "fresh")
    assert.equal(current.session?.sessionId, "facebook-session")
    assert.equal(closed, false)
    assert.equal(internal.eventStreams.has(marker), true)
    internal.eventStreams.delete(marker)
  } finally {
    internal.capabilities = originalCapabilities
    await client.close()
  }
})

test("ManagedRuntime stop is idempotent", async () => {
  const runtime = await ManagedRuntime.start({runtimePath})
  assert.ok(runtime.pid)
  await runtime.stop()
  await runtime.stop()
})
