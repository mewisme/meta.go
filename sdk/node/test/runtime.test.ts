import assert from "node:assert/strict"
import {execFileSync} from "node:child_process"
import {mkdtempSync, rmSync} from "node:fs"
import {tmpdir} from "node:os"
import {resolve} from "node:path"
import {fileURLToPath} from "node:url"
import test, {after, before} from "node:test"
import {ManagedRuntime, MetaClient, MetaRpcError, UnsupportedCapabilityError} from "../src/index.js"

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
    assert.throws(() => client.requireCapability("future.missing"), UnsupportedCapabilityError)
    const created = await client.createSession({c_user: "42", xs: "test"}, {eventBuffer: 8, timeoutMs: 5000})
    assert.ok(created.sessionId)
    const health = await client.getSessionHealth(created.sessionId)
    assert.ok(health.health)
    await client.closeSession(created.sessionId)
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

test("ManagedRuntime stop is idempotent", async () => {
  const runtime = await ManagedRuntime.start({runtimePath})
  assert.ok(runtime.pid)
  await runtime.stop()
  await runtime.stop()
})
