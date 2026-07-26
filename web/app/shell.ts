// The portal shell: brand header, tool sidebar, sign-in gate, and the hash
// router that mounts/disposes ToolSurfaces. Tool-agnostic — it renders whatever
// registry.ts lists, so lagotto-ts et al. drop in with no shell change.
import { surfaces, findSurface } from "./surfaces/registry.js";
import type { PortalConfig, SurfaceContext, Disposable } from "./surfaces/types.js";
import type { SessionController } from "./session.js";
import { startSignIn } from "./auth/globus-login.js";

export class Shell {
  private navEl!: HTMLElement;
  private mainEl!: HTMLElement;
  private bannerEl!: HTMLElement;
  private current: { id: string; disposable: Disposable } | null = null;

  constructor(
    private root: HTMLElement,
    private session: SessionController,
    private config: PortalConfig,
  ) {}

  /** Build the chrome, wire the router, and route to the initial hash. */
  start(): void {
    this.render();
    window.addEventListener("hashchange", () => void this.route());
    this.session.onExpiry((state) => this.showExpiry(state));
    void this.route();
  }

  private render(): void {
    this.root.innerHTML = `
      <header class="portal-header">
        <a class="portal-brand" href="/">
          <span class="logo-spore">spore</span><span class="logo-host">host</span>
        </a>
        <div class="portal-account"></div>
      </header>
      <div class="portal-banner" hidden></div>
      <div class="portal-body">
        <nav class="portal-nav" aria-label="Tools"></nav>
        <main class="portal-main" id="surface-host"></main>
      </div>`;
    this.navEl = this.root.querySelector(".portal-nav")!;
    this.mainEl = this.root.querySelector(".portal-main")!;
    this.bannerEl = this.root.querySelector(".portal-banner")!;
    this.renderNav();
    this.renderAccount();
  }

  private renderNav(): void {
    const active = this.currentId();
    this.navEl.innerHTML = surfaces
      .map(
        (s) =>
          `<a class="portal-tool${s.id === active ? " active" : ""}" href="#/${s.id}"
             style="--accent: var(${s.accent})">${escapeHtml(s.label)}</a>`,
      )
      .join("");
  }

  private renderAccount(): void {
    const el = this.root.querySelector<HTMLElement>(".portal-account")!;
    if (this.session.signedIn) {
      el.innerHTML = `<span class="acct">account <b>${escapeHtml(
        this.session.accountId ?? "?",
      )}</b></span> <button class="portal-signout">sign out</button>`;
      el.querySelector(".portal-signout")!.addEventListener("click", () => {
        this.session.clear();
        location.hash = "";
        this.render();
        void this.route();
      });
    } else {
      el.innerHTML = `<button class="portal-signin">sign in</button>`;
      el.querySelector(".portal-signin")!.addEventListener("click", () => {
        void startSignIn(this.config);
      });
    }
  }

  private currentId(): string {
    const id = location.hash.replace(/^#\/?/, "").split("/")[0];
    return id || surfaces[0]?.id || "";
  }

  private async route(): Promise<void> {
    const id = this.currentId();
    const surface = findSurface(id);
    // Dispose the outgoing surface first.
    if (this.current && this.current.id !== id) {
      this.current.disposable.dispose();
      this.current = null;
    }
    this.renderNav();

    if (!surface) {
      this.mainEl.innerHTML = `<div class="portal-empty">Unknown tool: ${escapeHtml(id)}</div>`;
      return;
    }
    if (surface.requiresAuth && !this.session.signedIn) {
      this.renderSignInGate(surface.label);
      return;
    }
    if (this.current?.id === id) return; // already mounted

    this.mainEl.innerHTML = "";
    const ctx: SurfaceContext = {
      session: this.session,
      config: this.config,
      navigate: (sid) => {
        location.hash = `#/${sid}`;
      },
    };
    try {
      const disposable = await surface.mount(this.mainEl, ctx);
      this.current = { id, disposable };
    } catch (err) {
      this.mainEl.innerHTML = `<div class="portal-error">Failed to load ${escapeHtml(
        surface.label,
      )}: ${escapeHtml((err as Error).message)}</div>`;
    }
  }

  private renderSignInGate(label: string): void {
    this.mainEl.innerHTML = `
      <div class="portal-gate">
        <h2>Sign in to use ${escapeHtml(label)}</h2>
        <p>spore.host federates your institutional identity (via Globus) into your
           own AWS account — credentials stay in this browser tab.</p>
        <button class="portal-signin-lg">Sign in with Globus</button>
      </div>`;
    this.mainEl.querySelector(".portal-signin-lg")!.addEventListener("click", () => {
      void startSignIn(this.config);
    });
  }

  private showExpiry(state: "warning" | "expired"): void {
    this.bannerEl.hidden = false;
    if (state === "warning") {
      this.bannerEl.className = "portal-banner warn";
      this.bannerEl.textContent = "Your session expires soon — sign in again to stay connected.";
    } else {
      this.bannerEl.className = "portal-banner expired";
      this.bannerEl.innerHTML = `Session expired. <button class="portal-reauth">Sign in again</button>`;
      this.bannerEl.querySelector(".portal-reauth")!.addEventListener("click", () => {
        void startSignIn(this.config);
      });
    }
  }
}

function escapeHtml(s: string): string {
  return s.replace(
    /[&<>"']/g,
    (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" })[c]!,
  );
}
