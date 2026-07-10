import { existsSync, mkdirSync, rmSync } from "node:fs";
import { join, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { spawnSync } from "node:child_process";

const root = resolve(fileURLToPath(new URL("..", import.meta.url)));
const outDir = join(root, "out");
const exeName = process.platform === "win32" ? "riftdtl.exe" : "riftdtl";
const output = join(outDir, exeName);
const args = new Set(process.argv.slice(2));
const go = process.env.GO_BIN ?? "go";

if (args.has("--clean")) {
  rmSync(outDir, { recursive: true, force: true });
}
mkdirSync(outDir, { recursive: true });

const build = spawnSync(go, ["build", "-trimpath", "-o", output, "./src"], {
  cwd: root,
  encoding: "utf8",
  stdio: "inherit",
  shell: false,
});

if (build.status !== 0) {
  if (args.has("--reuse-existing") && existsSync(output)) {
    console.warn(`Go build unavailable; reusing existing binary: ${output}`);
    console.log(output);
    process.exit(0);
  }
  console.error(`Go build failed. Install Go or set GO_BIN to the compiler path. Attempted: ${go}`);
  process.exit(build.status ?? 1);
}

console.log(output);
