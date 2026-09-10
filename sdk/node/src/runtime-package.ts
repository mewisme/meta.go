import {constants, access} from "node:fs/promises"
import {createRequire} from "node:module"
import {RuntimeLaunchError} from "./errors.js"

export type RuntimePlatform = "linux" | "darwin" | "win32"
export type RuntimeArch = "x64" | "arm64"
export interface RuntimeTarget { platform: RuntimePlatform; arch: RuntimeArch; filename: "meta-runtime" | "meta-runtime.exe" }

const resolvePackage = (specifier: string): string => createRequire(import.meta.url).resolve(specifier)

export function runtimeTarget(platform: NodeJS.Platform = process.platform, arch: string = process.arch): RuntimeTarget {
  if (platform !== "linux" && platform !== "darwin" && platform !== "win32") throw new RuntimeLaunchError(`unsupported runtime platform ${platform}`)
  if (arch !== "x64" && arch !== "arm64") throw new RuntimeLaunchError(`unsupported runtime architecture ${arch}`)
  return {platform, arch, filename: platform === "win32" ? "meta-runtime.exe" : "meta-runtime"}
}

export function runtimePackageName(target: RuntimeTarget): string { return `@meewmeew/meta-runtime-${target.platform}-${target.arch}` }

export async function resolveBundledRuntime(target: RuntimeTarget, resolve: (specifier: string) => string = resolvePackage): Promise<string | undefined> {
  const packageName = runtimePackageName(target)
  let runtimePath: string
  try {
    runtimePath = resolve(`${packageName}/runtime`)
  } catch (error) {
    if (moduleMissing(error)) return undefined
    throw new RuntimeLaunchError(`failed to resolve bundled runtime package ${packageName}`, {cause: error})
  }
  try {
    await access(runtimePath, constants.X_OK)
  } catch (error) {
    throw new RuntimeLaunchError(`bundled runtime package ${packageName} does not contain an executable runtime`, {cause: error})
  }
  return runtimePath
}

function moduleMissing(error: unknown): boolean {
  if (!error || typeof error !== "object" || !("code" in error)) return false
  const code = String((error as {code?: unknown}).code)
  return code === "MODULE_NOT_FOUND" || code === "ERR_MODULE_NOT_FOUND"
}
