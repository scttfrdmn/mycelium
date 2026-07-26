// The strata catalog surface: browse the software "formations" (curated
// environment stacks — R, Python ML, HPC/MPI, genomics, CUDA) the shared
// dashboard-api publishes. This is the first Slice 5 surface: it consumes the
// multi-tenant API (api.spore.host) authenticated by the signed-in user's
// federated AWS creds in an X-AWS-Credentials header — the same path teams,
// cost-history, and Slack will use next.
//
// The catalog itself is static/shared (not per-account), so this is the
// lowest-risk exercise of the X-AWS-Credentials plumbing end to end.
import type { AwsCreds } from "@spore-host/spawn-ts/auth";
import type { Disposable, SurfaceContext, ToolSurface } from "./types.js";

interface Formation {
  name: string;
  display_name: string;
  description: string;
}

interface CatalogResponse {
  success: boolean;
  formations?: Formation[];
  error?: string;
}

/**
 * Base64-encode the session's federated creds for the X-AWS-Credentials header
 * the dashboard-api validates via STS GetCallerIdentity. Creds stay in memory
 * (from the SessionController) — never persisted.
 */
function credentialsHeader(creds: AwsCreds): string {
  const json = JSON.stringify({
    accessKeyId: creds.accessKeyId,
    secretAccessKey: creds.secretAccessKey,
    sessionToken: creds.sessionToken,
  });
  // btoa is fine here: the JSON is ASCII (base64/hex AWS cred fields).
  return btoa(json);
}

export const catalogSurface: ToolSurface = {
  id: "catalog",
  label: "Software catalog",
  accent: "--strata",
  requiresAuth: true,

  async mount(host: HTMLElement, ctx: SurfaceContext): Promise<Disposable> {
    const creds = ctx.session.getCreds();
    if (!creds) throw new Error("catalog surface mounted without a session");

    const root = document.createElement("div");
    root.className = "catalog-surface";
    root.innerHTML = `
      <div class="catalog-head">
        <h2>Software catalog</h2>
        <p class="catalog-hint">Curated environment formations you can launch onto an
          instance. Served by the shared dashboard-api over your federated session.</p>
      </div>
      <div class="catalog-results" aria-live="polite"><div class="catalog-status">loading…</div></div>`;
    host.appendChild(root);

    const results = root.querySelector<HTMLElement>(".catalog-results")!;
    const controller = new AbortController();

    async function load(): Promise<void> {
      results.innerHTML = `<div class="catalog-status">loading…</div>`;
      try {
        const resp = await fetch(`${ctx.config.apiBase}/api/strata/catalog`, {
          method: "GET",
          headers: { "X-AWS-Credentials": credentialsHeader(creds!) },
          signal: controller.signal,
        });
        if (!resp.ok) {
          const detail = resp.status === 401 ? "authentication failed" : `HTTP ${resp.status}`;
          results.innerHTML = `<div class="catalog-status error">Couldn't load the catalog (${escapeHtml(detail)}).</div>`;
          return;
        }
        const body = (await resp.json()) as CatalogResponse;
        if (!body.success || !body.formations) {
          results.innerHTML = `<div class="catalog-status error">${escapeHtml(body.error ?? "no catalog returned")}</div>`;
          return;
        }
        renderCatalog(results, body.formations);
      } catch (err) {
        if (controller.signal.aborted) return; // surface torn down mid-flight
        results.innerHTML = `<div class="catalog-status error">${escapeHtml((err as Error).message)}</div>`;
      }
    }

    // Reload if creds expire+refresh isn't wired yet — for now just surface it.
    const offExpiry = ctx.session.onExpiry((state) => {
      if (state === "expired") {
        results.innerHTML = `<div class="catalog-status error">Session expired — sign in again to reload the catalog.</div>`;
      }
    });

    void load();

    return {
      dispose() {
        offExpiry();
        controller.abort();
        root.remove();
      },
    };
  },
};

function renderCatalog(host: HTMLElement, formations: Formation[]): void {
  if (formations.length === 0) {
    host.innerHTML = `<div class="catalog-status">the catalog is empty</div>`;
    return;
  }
  const cards = formations
    .map(
      (f) => `
      <div class="catalog-card">
        <div class="catalog-card-head">
          <span class="catalog-name">${escapeHtml(f.display_name)}</span>
          <code class="catalog-slug">${escapeHtml(f.name)}</code>
        </div>
        <p class="catalog-desc">${escapeHtml(f.description)}</p>
      </div>`,
    )
    .join("");
  host.innerHTML = `<div class="catalog-count">${formations.length} formation${formations.length === 1 ? "" : "s"}</div>${cards}`;
}

function escapeHtml(s: string): string {
  return s.replace(
    /[&<>"']/g,
    (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" })[c]!,
  );
}
