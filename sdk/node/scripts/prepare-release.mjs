import {readFile, writeFile} from "node:fs/promises"
import {dirname, join, resolve} from "node:path"
import {fileURLToPath} from "node:url"

const defaultSdkDir = resolve(dirname(fileURLToPath(import.meta.url)), "..")
const targets = ["darwin-arm64", "darwin-x64", "linux-arm64", "linux-x64", "win32-arm64", "win32-x64"]
const version = process.argv[2]
const sdkDirFlag = process.argv.indexOf("--sdk-dir")
const sdkDir = sdkDirFlag >= 0 ? resolve(process.argv[sdkDirFlag + 1] ?? "") : defaultSdkDir

if (!version || !/^\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?$/.test(version)) throw new Error(`invalid package version ${version ?? ""}`)

const rootPath = join(sdkDir, "package.json")
const root = JSON.parse(await readFile(rootPath, "utf8"))
root.version = version
root.optionalDependencies = Object.fromEntries(targets.map(target => [`@meewmeew/meta-runtime-${target}`, version]))
await writeJSON(rootPath, root)

for (const target of targets) {
  const path = join(sdkDir, "runtime-packages", target, "package.json")
  const packageJSON = JSON.parse(await readFile(path, "utf8"))
  packageJSON.version = version
  await writeJSON(path, packageJSON)
}

async function writeJSON(path, value) { await writeFile(path, `${JSON.stringify(value, null, 2)}\n`) }
