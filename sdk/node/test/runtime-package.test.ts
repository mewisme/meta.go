import assert from "node:assert/strict"
import {execFileSync} from "node:child_process"
import {chmod, cp, mkdtemp, readFile, rm, writeFile} from "node:fs/promises"
import {tmpdir} from "node:os"
import {join, resolve} from "node:path"
import {fileURLToPath} from "node:url"
import test from "node:test"
import {resolveBundledRuntime, runtimePackageName, runtimeTarget, type RuntimeArch, type RuntimePlatform} from "../src/runtime-package.js"

const sdkDir = fileURLToPath(new URL("../", import.meta.url))
const targets: Array<[RuntimePlatform, RuntimeArch, string]> = [
  ["linux", "x64", "@meewmeew/meta-runtime-linux-x64"],
  ["linux", "arm64", "@meewmeew/meta-runtime-linux-arm64"],
  ["darwin", "x64", "@meewmeew/meta-runtime-darwin-x64"],
  ["darwin", "arm64", "@meewmeew/meta-runtime-darwin-arm64"],
  ["win32", "x64", "@meewmeew/meta-runtime-win32-x64"],
  ["win32", "arm64", "@meewmeew/meta-runtime-win32-arm64"],
]

test("runtime package mapping covers every supported target", async () => {
  for (const [platform, arch, name] of targets) {
    const target = runtimeTarget(platform, arch)
    assert.equal(runtimePackageName(target), name)
    assert.equal(target.filename, platform === "win32" ? "meta-runtime.exe" : "meta-runtime")
    const packageJSON = JSON.parse(await readFile(resolve(sdkDir, "runtime-packages", `${platform}-${arch}`, "package.json"), "utf8")) as {name: string; version: string; os: string[]; cpu: string[]; bin: Record<string, string>; scripts?: unknown; dependencies?: unknown}
    assert.equal(packageJSON.name, name)
    assert.equal(packageJSON.version, "0.0.0")
    assert.deepEqual(packageJSON.os, [platform])
    assert.deepEqual(packageJSON.cpu, [arch])
    assert.equal(packageJSON.bin["meta-runtime"], platform === "win32" ? "./meta-runtime.exe" : "./meta-runtime")
    assert.equal(packageJSON.scripts, undefined)
    assert.equal(packageJSON.dependencies, undefined)
  }
})

test("resolveBundledRuntime resolves the stable runtime export", async () => {
  const dir = await mkdtemp(join(tmpdir(), "meta-runtime-package-"))
  const runtime = join(dir, "meta-runtime")
  try {
    await writeFile(runtime, "runtime")
    await chmod(runtime, 0o755)
    const target = runtimeTarget("linux", "x64")
    const resolved = await resolveBundledRuntime(target, specifier => {
      assert.equal(specifier, "@meewmeew/meta-runtime-linux-x64/runtime")
      return runtime
    })
    assert.equal(resolved, runtime)
  } finally {
    await rm(dir, {recursive: true, force: true})
  }
})

test("release preparation pins all runtime packages to the root SDK version", async () => {
  const temp = await mkdtemp(join(tmpdir(), "meta-node-release-"))
  try {
    await cp(resolve(sdkDir, "package.json"), join(temp, "package.json"))
    await cp(resolve(sdkDir, "runtime-packages"), join(temp, "runtime-packages"), {recursive: true})
    execFileSync(process.execPath, [resolve(sdkDir, "scripts", "prepare-release.mjs"), "0.260911.0", "--sdk-dir", temp])
    const root = JSON.parse(await readFile(join(temp, "package.json"), "utf8")) as {version: string; optionalDependencies: Record<string, string>}
    assert.equal(root.version, "0.260911.0")
    assert.equal(Object.keys(root.optionalDependencies).length, targets.length)
    for (const [, , name] of targets) assert.equal(root.optionalDependencies[name], "0.260911.0")
    for (const [platform, arch] of targets) {
      const packageJSON = JSON.parse(await readFile(join(temp, "runtime-packages", `${platform}-${arch}`, "package.json"), "utf8")) as {version: string}
      assert.equal(packageJSON.version, "0.260911.0")
    }
  } finally {
    await rm(temp, {recursive: true, force: true})
  }
})
