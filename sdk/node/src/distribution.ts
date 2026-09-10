import {constants, access} from "node:fs/promises"
import {delimiter, join} from "node:path"
import {RuntimeLaunchError} from "./errors.js"
import {resolveBundledRuntime, runtimeTarget} from "./runtime-package.js"

export interface RuntimeDistributionOptions { runtimePath?: string; env?: NodeJS.ProcessEnv }

export async function resolveRuntimePath(options: RuntimeDistributionOptions = {}): Promise<string> {
  if (options.runtimePath) return options.runtimePath
  const env = {...process.env, ...options.env}
  if (env.META_RUNTIME_PATH) return env.META_RUNTIME_PATH
  const target = runtimeTarget()
  const bundled = await resolveBundledRuntime(target)
  if (bundled) return bundled
  const installed = await findOnPath(target.filename, env.PATH)
  if (installed) return installed
  throw new RuntimeLaunchError(`${target.filename} was not found in the bundled runtime package or on PATH`)
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
