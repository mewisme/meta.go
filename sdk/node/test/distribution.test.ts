import assert from "node:assert/strict"
import {createHash} from "node:crypto"
import {mkdtemp, readFile, rm} from "node:fs/promises"
import {createServer} from "node:http"
import {tmpdir} from "node:os"
import {join} from "node:path"
import test from "node:test"
import {resolveRuntimePath, RuntimeLaunchError} from "../src/index.js"

const os = process.platform === "win32" ? "windows" : process.platform === "darwin" ? "darwin" : "linux"
const arch = process.arch === "arm64" ? "arm64" : "amd64"

test("resolveRuntimePath downloads, verifies and reuses a cached runtime", async () => {
  const version = "test"
  const asset = `meta-runtime_${version}_${os}_${arch}${os === "windows" ? ".exe" : ""}`
  const binary = Buffer.from("runtime-test")
  const checksum = createHash("sha256").update(binary).digest("hex")
  let requests = 0
  const server = createServer((request, response) => {
    requests++
    if (request.url?.endsWith("_checksums.txt")) response.end(`${checksum}  ${asset}\n`)
    else if (request.url?.endsWith(`/${asset}`)) response.end(binary)
    else { response.statusCode = 404; response.end() }
  })
  await new Promise<void>(resolve => server.listen(0, "127.0.0.1", resolve))
  const address = server.address()
  assert.ok(address && typeof address !== "string")
  const cache = await mkdtemp(join(tmpdir(), "meta-runtime-cache-"))
  const options = {runtimeVersion: version, runtimeCacheDir: cache, releaseBaseUrl: `http://127.0.0.1:${address.port}`, env: {PATH: ""}}
  try {
    const first = await resolveRuntimePath(options)
    assert.equal((await readFile(first)).toString(), binary.toString())
    assert.equal(requests, 2)
    const second = await resolveRuntimePath(options)
    assert.equal(second, first)
    assert.equal(requests, 2)
  } finally {
    await new Promise<void>((resolve, reject) => server.close(error => error ? reject(error) : resolve()))
    await rm(cache, {recursive: true, force: true})
  }
})

test("resolveRuntimePath rejects a runtime with a mismatched checksum", async () => {
  const version = "bad"
  const asset = `meta-runtime_${version}_${os}_${arch}${os === "windows" ? ".exe" : ""}`
  const server = createServer((request, response) => {
    if (request.url?.endsWith("_checksums.txt")) response.end(`${"0".repeat(64)}  ${asset}\n`)
    else response.end("tampered")
  })
  await new Promise<void>(resolve => server.listen(0, "127.0.0.1", resolve))
  const address = server.address()
  assert.ok(address && typeof address !== "string")
  const cache = await mkdtemp(join(tmpdir(), "meta-runtime-cache-"))
  try {
    await assert.rejects(resolveRuntimePath({runtimeVersion: version, runtimeCacheDir: cache, releaseBaseUrl: `http://127.0.0.1:${address.port}`, env: {PATH: ""}}), RuntimeLaunchError)
  } finally {
    await new Promise<void>((resolve, reject) => server.close(error => error ? reject(error) : resolve()))
    await rm(cache, {recursive: true, force: true})
  }
})
