// The truffle surface: a standalone EC2 instance-type catalog search, offline
// and auth-free. truffle-ts ships pure logic + a bundled catalog (no UI), so this
// mounts a thin search box over find() and renders the ranked results + match
// reasons. No AWS creds, no network — browse/compare before you ever sign in.
import { find, type FindResult } from "@spore-host/truffle-ts";
import type { Disposable, SurfaceContext, ToolSurface } from "./types.js";

export const truffleSurface: ToolSurface = {
  id: "truffle",
  label: "Find instances",
  accent: "--truffle",
  requiresAuth: false,

  async mount(host: HTMLElement, _ctx: SurfaceContext): Promise<Disposable> {
    const root = document.createElement("div");
    root.className = "truffle-surface";
    root.innerHTML = `
      <div class="truffle-search">
        <h2>Find an instance type</h2>
        <p class="truffle-hint">Natural language — e.g. <code>gpu with 80gb for training</code>,
          <code>32 vcpus arm</code>, <code>cheapest 64gb</code>. Offline catalog (${"as of 2026-01"}).</p>
        <form class="truffle-form">
          <input class="truffle-q" type="search" placeholder="describe what you need…" autocomplete="off" />
          <button type="submit">Search</button>
        </form>
      </div>
      <div class="truffle-results" aria-live="polite"></div>`;
    host.appendChild(root);

    const form = root.querySelector<HTMLFormElement>(".truffle-form")!;
    const input = root.querySelector<HTMLInputElement>(".truffle-q")!;
    const results = root.querySelector<HTMLElement>(".truffle-results")!;

    // Guard against out-of-order async results if the user searches rapidly.
    let seq = 0;
    async function run(query: string): Promise<void> {
      const mine = ++seq;
      if (!query.trim()) {
        results.innerHTML = "";
        return;
      }
      results.innerHTML = `<div class="truffle-status">searching…</div>`;
      try {
        const found = await find(query);
        if (mine !== seq) return; // a newer search superseded this one
        renderResults(results, found);
      } catch (err) {
        if (mine !== seq) return;
        results.innerHTML = `<div class="truffle-status error">${escapeHtml((err as Error).message)}</div>`;
      }
    }

    const onSubmit = (e: Event) => {
      e.preventDefault();
      void run(input.value);
    };
    form.addEventListener("submit", onSubmit);

    return {
      dispose() {
        form.removeEventListener("submit", onSubmit);
        root.remove();
      },
    };
  },
};

function renderResults(host: HTMLElement, found: FindResult[]): void {
  if (found.length === 0) {
    host.innerHTML = `<div class="truffle-status">no matches — try relaxing the query</div>`;
    return;
  }
  const rows = found
    .slice(0, 50)
    .map((r) => {
      const i = r.instance;
      const gib = (i.memoryMib / 1024).toFixed(i.memoryMib % 1024 === 0 ? 0 : 1);
      const price = i.onDemandPrice != null ? `$${i.onDemandPrice.toFixed(4)}/hr${i.estimatedPrice ? "*" : ""}` : "—";
      const gpu = i.gpus ? `${i.gpus}× ${escapeHtml(i.gpuModel ?? "GPU")}` : "";
      const reasons = r.reasons.map((x) => `<li>${escapeHtml(x)}</li>`).join("");
      return `
        <div class="truffle-row">
          <div class="truffle-row-head">
            <span class="truffle-type">${escapeHtml(i.instanceType)}</span>
            <span class="truffle-specs">${i.vcpus} vCPU · ${gib} GiB · ${escapeHtml(i.architecture)}${gpu ? " · " + gpu : ""}</span>
            <span class="truffle-price">${price}</span>
          </div>
          <ul class="truffle-reasons">${reasons}</ul>
        </div>`;
    })
    .join("");
  const note = found.some((r) => r.instance.estimatedPrice) ? `<div class="truffle-note">* estimated price (type not in the live-pulled region)</div>` : "";
  host.innerHTML = `<div class="truffle-count">${found.length} match${found.length === 1 ? "" : "es"}${found.length > 50 ? " (showing 50)" : ""}</div>${rows}${note}`;
}

function escapeHtml(s: string): string {
  return s.replace(
    /[&<>"']/g,
    (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" })[c]!,
  );
}
