import {spawnSync} from "node:child_process"
import {chmod, copyFile, mkdir, mkdtemp, readFile, rm, writeFile} from "node:fs/promises"
import {tmpdir} from "node:os"
import {dirname, isAbsolute, join, resolve} from "node:path"
import {fileURLToPath} from "node:url"

const sdkDir = resolve(dirname(fileURLToPath(import.meta.url)), "..")
const repoRoot = resolve(sdkDir, "../..")
const targets = new Set(["linux-x64", "linux-arm64", "darwin-x64", "darwin-arm64", "win32-x64", "win32-arm64"])
const args = parseArgs(process.argv.slice(2))
const target = required(args, "target")
const version = required(args, "version")
const binary = resolve(required(args, "binary"))
const outDir = resolve(required(args, "out-dir"))

if (!targets.has(target)) throw new Error(`unsupported runtime package target ${target}`)
validateVersion(version)

const templateDir = join(sdkDir, "runtime-packages", target)
const template = JSON.parse(await readFile(join(templateDir, "package.json"), "utf8"))
const filename = target.startsWith("win32-") ? "meta-runtime.exe" : "meta-runtime"
const stage = await mkdtemp(join(tmpdir(), "meta-node-runtime-"))

try {
  await mkdir(outDir, {recursive: true})
  await writeFile(join(stage, "package.json"), `${JSON.stringify({...template, version}, null, 2)}\n`)
  await copyFile(binary, join(stage, filename))
  await chmod(join(stage, filename), 0o755)
  await copyFile(join(sdkDir, "runtime-packages", "README.md"), join(stage, "README.md"))
  await copyFile(join(repoRoot, "LICENSE"), join(stage, "LICENSE"))
  const packed = spawnSync("pnpm", ["pack", "--pack-destination", outDir, "--silent"], {cwd: stage, encoding: "utf8"})
  if (packed.status !== 0) throw new Error(`pnpm pack failed: ${(packed.stderr || packed.stdout).trim()}`)
  const output = packed.stdout.trim().split(/\r?\n/).at(-1)
  if (!output) throw new Error("pnpm pack returned no artifact path")
  console.log(isAbsolute(output) ? output : resolve(stage, output))
} finally {
  await rm(stage, {recursive: true, force: true})
}

function parseArgs(values) {
  const result = new Map()
  for (let index = 0; index < values.length; index += 2) {
    const key = values[index]
    const value = values[index + 1]
    if (!key?.startsWith("--") || value === undefined) throw new Error(`invalid argument ${key ?? ""}`)
    result.set(key.slice(2), value)
  }
  return result
}

function required(values, key) {
  const value = values.get(key)
  if (!value) throw new Error(`--${key} is required`)
  return value
}

function validateVersion(version) {
  if (!/^\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?$/.test(version)) throw new Error(`invalid package version ${version}`)
}
