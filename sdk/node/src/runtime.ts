import {spawn, type ChildProcess} from "node:child_process"
import {basename} from "node:path"
import {setTimeout as delay} from "node:timers/promises"
import {ProtocolMismatchError, RuntimeLaunchError} from "./errors.js"

export const PROTOCOL_MAJOR = 1
const maxBootstrapBytes = 64 << 10
const maxStderrBytes = 16 << 10

export interface RuntimeBootstrap { endpoint: string; token: string; protocol: number }
export interface ManagedRuntimeOptions { runtimePath?: string; listen?: string; token?: string; env?: NodeJS.ProcessEnv; bootstrapTimeoutMs?: number; stopTimeoutMs?: number }

export class ManagedRuntime {
  readonly endpoint: string
  readonly token: string
  readonly protocol: number
  readonly pid: number | undefined
  private readonly child: ChildProcess
  private readonly stopTimeoutMs: number
  private stopped = false

  private constructor(child: ChildProcess, bootstrap: RuntimeBootstrap, stopTimeoutMs: number) {
    this.child = child
    this.endpoint = bootstrap.endpoint
    this.token = bootstrap.token
    this.protocol = bootstrap.protocol
    this.pid = child.pid
    this.stopTimeoutMs = stopTimeoutMs
  }

  static async start(options: ManagedRuntimeOptions = {}): Promise<ManagedRuntime> {
    const runtimePath = options.runtimePath ?? process.env.META_RUNTIME_PATH ?? (process.platform === "win32" ? "meta-runtime.exe" : "meta-runtime")
    const child = spawn(runtimePath, ["--listen", options.listen ?? "127.0.0.1:0"], {env: {...process.env, ...options.env, ...(options.token ? {META_RUNTIME_TOKEN: options.token} : {})}, stdio: ["ignore", "pipe", "pipe"], windowsHide: true})
    const stderr: Buffer[] = []
    let stderrBytes = 0
    child.stderr?.on("data", (chunk: Buffer) => {
      if (stderrBytes >= maxStderrBytes) return
      const kept = chunk.subarray(0, maxStderrBytes - stderrBytes)
      stderr.push(kept)
      stderrBytes += kept.length
    })
    try {
      const bootstrap = await readBootstrap(child, options.bootstrapTimeoutMs ?? 10_000, () => Buffer.concat(stderr).toString("utf8").trim())
      if (bootstrap.protocol !== PROTOCOL_MAJOR) {
        child.kill("SIGTERM")
        throw new ProtocolMismatchError(PROTOCOL_MAJOR, bootstrap.protocol)
      }
      return new ManagedRuntime(child, bootstrap, options.stopTimeoutMs ?? 5_000)
    } catch (error) {
      if (child.exitCode === null && child.signalCode === null) child.kill("SIGTERM")
      if (error instanceof ProtocolMismatchError || error instanceof RuntimeLaunchError) throw error
      throw new RuntimeLaunchError(`failed to start ${basename(runtimePath)}`, {cause: error})
    }
  }

  async stop(): Promise<void> {
    if (this.stopped) return
    this.stopped = true
    if (this.child.exitCode !== null || this.child.signalCode !== null) return
    const exited = new Promise<void>(resolve => this.child.once("exit", () => resolve()))
    this.child.kill("SIGTERM")
    if (await Promise.race([exited.then(() => true), delay(this.stopTimeoutMs).then(() => false)])) return
    this.child.kill("SIGKILL")
    await exited
  }
}

async function readBootstrap(child: ChildProcess, timeoutMs: number, stderr: () => string): Promise<RuntimeBootstrap> {
  const stdout = child.stdout
  if (!stdout) throw new RuntimeLaunchError("runtime stdout is unavailable")
  return new Promise<RuntimeBootstrap>((resolve, reject) => {
    let data = Buffer.alloc(0)
    let settled = false
    const finish = (error?: unknown, value?: RuntimeBootstrap) => {
      if (settled) return
      settled = true
      clearTimeout(timer)
      stdout.off("data", onData)
      child.off("error", onError)
      child.off("exit", onExit)
      if (error) reject(error)
      else resolve(value!)
    }
    const onData = (chunk: Buffer) => {
      data = Buffer.concat([data, chunk])
      if (data.length > maxBootstrapBytes) return finish(new RuntimeLaunchError("runtime bootstrap line is too large"))
      const newline = data.indexOf(0x0a)
      if (newline < 0) return
      const line = data.subarray(0, newline).toString("utf8").trim()
      try {
        const parsed = JSON.parse(line) as Partial<RuntimeBootstrap>
        if (!parsed.endpoint || !parsed.token || typeof parsed.protocol !== "number" || !Number.isInteger(parsed.protocol)) throw new Error("invalid bootstrap fields")
        finish(undefined, {endpoint: parsed.endpoint, token: parsed.token, protocol: parsed.protocol})
      } catch (error) {
        finish(new RuntimeLaunchError("runtime emitted invalid bootstrap JSON", {cause: error}))
      }
    }
    const onError = (error: Error) => finish(new RuntimeLaunchError(`runtime process failed: ${error.message}`, {cause: error}))
    const onExit = (code: number | null, signal: NodeJS.Signals | null) => finish(new RuntimeLaunchError(`runtime exited before bootstrap (${signal ?? code ?? "unknown"})${stderr() ? `: ${stderr()}` : ""}`))
    const timer = setTimeout(() => finish(new RuntimeLaunchError(`runtime bootstrap timed out after ${timeoutMs}ms${stderr() ? `: ${stderr()}` : ""}`)), timeoutMs)
    stdout.on("data", onData)
    child.once("error", onError)
    child.once("exit", onExit)
  })
}
