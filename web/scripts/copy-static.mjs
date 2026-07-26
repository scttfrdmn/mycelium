// Copy the no-build marketing pages + shared static dirs into dist/ verbatim,
// after `vite build` has emitted the portal SPA under dist/app/. These pages have
// no module graph — Vite would only risk mangling them (its strict HTML parser
// rejects the literal `&` in their URLs), so they bypass the bundler entirely.
import { cp, mkdir } from "node:fs/promises";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const web = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const dist = resolve(web, "dist");

const pages = ["index.html", "privacy.html", "terms.html", "library.html"];
const dirs = ["css", "assets"];

await mkdir(dist, { recursive: true });
for (const p of pages) {
  await cp(resolve(web, p), resolve(dist, p));
}
for (const d of dirs) {
  await cp(resolve(web, d), resolve(dist, d), { recursive: true });
}

// The BYOA onboarding CloudFormation template — served publicly so the portal's
// "Connect account" quick-create URL can point CloudFormation at it. Lives in the
// repo's deployment/ dir (one level above web/), copied to dist/cloudformation/.
const cfnSrc = resolve(web, "..", "deployment", "cloudformation", "portal-onboarding-role.yaml");
const cfnDest = resolve(dist, "cloudformation", "portal-onboarding-role.yaml");
await mkdir(dirname(cfnDest), { recursive: true });
await cp(cfnSrc, cfnDest);

console.log(`copied ${pages.length} pages + ${dirs.join(", ")} + onboarding template → dist/`);
