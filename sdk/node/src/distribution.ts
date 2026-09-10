import {createHash} from "node:crypto"
import {constants, access, chmod, mkdir, readFile, rename, rm, writeFile} from "node:fs/promises"
import {homedir} from "node:os"
import {delimiter, join} from "node:path"
import {RuntimeLaunchError} from "./errors.js"

const defaultReleaseBaseUrl = "https://github.com/mewisme/meta.go/releases/download"

export interface RuntimeDistributionOptions {
  runtimePath?: string
  runtimeVersion?: string
  runtimeCacheDir?: string
  releaseBaseUrl?: string
  downloadRuntime?: boolean
  env?: NodeJS.ProcessEnv
}

interface RuntimeTarget { os: "linux" | "darwin" | "windows"; arch: "amd64" | "arm64"; filename: string }

export async function resolveRuntimePath(options: RuntimeDistributionOptions = {}): Promise<string> {
  if (options.runtimePath) return options.runtimePath
  const env = {...process.env, ...options.env}
  if (env.META_RUNTIME_PATH) return env.META_RUNTIME_PATH
  const target = runtimeTarget()
  const installed = await findOnPath(target.filename, env.PATH)
  if (installed) return installed
  if (options.downloadRuntime === false) throw new RuntimeLaunchError(`${target.filename} was not found on PATH and runtime downloading is disabled`)
  const version = normalizeVersion(options.runtimeVersion ?? env.META_RUNTIME_VERSION ?? await packageVersion())
  const cacheRoot = options.runtimeCacheDir ?? env.META_RUNTIME_CACHE_DIR ?? defaultCacheDir(env)
  const cacheDir = join(cacheRoot, version, `${target.os}-${target.arch}`)
  const runtimePath = join(cacheDir, target.filename)
  const checksumPath = `${runtimePath}.sha256`
  if (await verifyCached(runtimePath, checksumPath)) return runtimePath
  await mkdir(cacheDir, {recursive: true})
  return downloadRuntime(version, target, runtimePath, checksumPath, options.releaseBaseUrl ?? env.META_RUNTIME_RELEASE_BASE_URL ?? defaultReleaseBaseUrl)
}

function runtimeTarget(): RuntimeTarget {
  const os = process.platform === "win32" ? "windows" : process.platform === "darwin" ? "darwin" : process.platform === "linux" ? "linux" : undefined
  const arch = process.arch === "x64" ? "amd64" : process.arch === "arm64" ? "arm64" : undefined
  if (!os || !arch) throw new RuntimeLaunchError(`unsupported runtime target ${process.platform}/${process.arch}`)
  return {os, arch, filename: os === "windows" ? "meta-runtime.exe" : "meta-runtime"}
}

async function findOnPath(filename: string, pathValue?: string): Promise<string | undefined> {
  if (!pathValue) return undefined
  for (const entry of pathValue.split(delimiter).filter(Boolean)) {
    const candidate = join(entry, filename)
    try {
      await access(candidate, constants.X_OK)
      return candidate
    } catch {}
  }
  return undefined
}

async function packageVersion(): Promise<string> {
  try {
    const value = JSON.parse(await readFile(new URL("../package.json", import.meta.url), "utf8")) as {version?: unknown}
    if (typeof value.version === "string" && value.version) return value.version
  } catch {}
  return "0.0.0"
}

function normalizeVersion(version: string): string {
  const normalized = version.startsWith("v") ? version.slice(1) : version
  if (!normalized || !/^[0-9A-Za-z][0-9A-Za-z.+-]*$/.test(normalized)) throw new RuntimeLaunchError(`invalid runtime version ${version}`)
  return normalized
}

function defaultCacheDir(env: NodeJS.ProcessEnv): string {
  if (env.META_RUNTIME_CACHE_DIR) return env.META_RUNTIME_CACHE_DIR
  if (process.platform === "win32" && env.LOCALAPPDATA) return join(env.LOCALAPPDATA, "meewmeew", "meta-runtime")
  if (env.XDG_CACHE_HOME) return join(env.XDG_CACHE_HOME, "meewmeew", "meta-runtime")
  return join(homedir(), ".cache", "meewmeew", "meta-runtime")
}

async function verifyCached(runtimePath: string, checksumPath: string): Promise<boolean> {
  try {
    const [binary, expected] = await Promise.all([readFile(runtimePath), readFile(checksumPath, "utf8")])
    return sha256(binary) === expected.trim().toLowerCase()
  } catch {
    return false
  }
}

async function downloadRuntime(version: string, target: RuntimeTarget, runtimePath: string, checksumPath: string, baseUrl: string): Promise<string> {
  const asset = `meta-runtime_${version}_${target.os}_${target.arch}${target.os === "windows" ? ".exe" : ""}`
  const root = `${baseUrl.replace(/\/$/, "")}/v${version}`
  const checksums = await fetchBytes(`${root}/meta_${version}_checksums.txt`)
  const expected = checksumFor(checksums.toString("utf8"), asset)
  const binary = await fetchBytes(`${root}/${asset}`)
  const actual = sha256(binary)
  if (actual !== expected) throw new RuntimeLaunchError(`checksum mismatch for ${asset}`)
  const temp = `${runtimePath}.${process.pid}.tmp`
  try {
    await writeFile(temp, binary, {mode: 0o755})
    if (target.os !== "windows") await chmod(temp, 0o755)
    await rm(runtimePath, {force: true})
    await rename(temp, runtimePath)
    await writeFile(checksumPath, `${expected}\n`)
  } finally {
    await rm(temp, {force: true}).catch(() => undefined)
  }
  return runtimePath
}

async function fetchBytes(url: string): Promise<Buffer> {
  let response: Response
  try {
    response = await fetch(url, {redirect: "follow"})
  } catch (error) {
    throw new RuntimeLaunchError(`failed to download runtime asset ${url}`, {cause: error})
  }
  if (!response.ok) throw new RuntimeLaunchError(`failed to download runtime asset ${url}: HTTP ${response.status}`)
  return Buffer.from(await response.arrayBuffer())
}

function checksumFor(manifest: string, asset: string): string {
  for (const line of manifest.split(/\r?\n/)) {
    const match = /^([a-fA-F0-9]{64})\s+\*?(.+)$/.exec(line.trim())
    if (match?.[2] === asset) return match[1]!.toLowerCase()
  }
  throw new RuntimeLaunchError(`checksum manifest does not contain ${asset}`)
}

function sha256(value: Buffer): string { return createHash("sha256").update(value).digest("hex") }
