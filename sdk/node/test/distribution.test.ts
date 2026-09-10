import assert from "node:assert/strict"
import {chmod, mkdtemp, rm, writeFile} from "node:fs/promises"
import {tmpdir} from "node:os"
import {delimiter, join} from "node:path"
import test from "node:test"
import {resolveRuntimePath, RuntimeLaunchError} from "../src/index.js"

const filename = process.platform === "win32" ? "meta-runtime.exe" : "meta-runtime"

test("resolveRuntimePath prefers explicit and environment paths", async () => {
  assert.equal(await resolveRuntimePath({runtimePath: "/explicit/runtime", env: {META_RUNTIME_PATH: "/env/runtime", PATH: ""}}), "/explicit/runtime")
  assert.equal(await resolveRuntimePath({env: {META_RUNTIME_PATH: "/env/runtime", PATH: ""}}), "/env/runtime")
})

test("resolveRuntimePath falls back to PATH without downloading", async () => {
  const dir = await mkdtemp(join(tmpdir(), "meta-runtime-path-"))
  const runtime = join(dir, filename)
  try {
    await writeFile(runtime, "runtime")
    await chmod(runtime, 0o755)
    assert.equal(await resolveRuntimePath({env: {META_RUNTIME_PATH: "", PATH: dir}}), runtime)
  } finally {
    await rm(dir, {recursive: true, force: true})
  }
})

test("resolveRuntimePath fails locally when no runtime is installed", async () => {
  await assert.rejects(resolveRuntimePath({env: {META_RUNTIME_PATH: "", PATH: delimiter === ";" ? "Z:\\missing" : "/missing"}}), error => error instanceof RuntimeLaunchError && !error.message.includes("download"))
})
