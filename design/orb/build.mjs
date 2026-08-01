// Rebuild web/assets/brand/spore-orb.riv from scene.json + parts/.
//
//   node design/orb/build.mjs            # writes web/assets/brand/spore-orb.riv
//   node design/orb/build.mjs /tmp/x.riv # writes elsewhere
//
// Requires the rive-mcp-server package on disk for its .riv writer (the same
// code path the riv_create MCP tool uses). Install it with:
//   npm i -g rive-mcp-server
// and set RIVE_MCP_DIR if it isn't under the default global prefix.
//
// The writer takes the scene inline and does NOT resolve pngPath itself — that
// is the MCP layer's job — so we read each part into `bytes` first.
import { readFileSync, writeFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";
import { execSync } from "node:child_process";

const here = dirname(fileURLToPath(import.meta.url));
const out = process.argv[2]
  ? resolve(process.argv[2])
  : resolve(here, "..", "..", "web", "assets", "brand", "spore-orb.riv");

const mcpDir =
  process.env.RIVE_MCP_DIR ??
  resolve(execSync("npm root -g", { encoding: "utf8" }).trim(), "rive-mcp-server");
const { createRiv } = await import(
  pathToFileURL(resolve(mcpDir, "dist", "rivWriter.js")).href
);

const scene = JSON.parse(readFileSync(resolve(here, "scene.json"), "utf8"));
for (const im of scene.images) {
  im.bytes = new Uint8Array(readFileSync(resolve(here, im.pngPath)));
}

const { bytes, warnings } = await createRiv(scene);
writeFileSync(out, bytes);
console.log(
  `${out} — ${bytes.length} bytes` +
    (warnings?.length ? `, warnings: ${JSON.stringify(warnings)}` : ""),
);
