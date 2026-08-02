// The portal shell: brand header, tool sidebar, sign-in gate, and the hash
// router that mounts/disposes ToolSurfaces. Tool-agnostic — it renders whatever
// registry.ts lists, so lagotto-ts et al. drop in with no shell change.
import { surfaces, findSurface } from "./surfaces/registry.js";
import type { PortalConfig, SurfaceContext, Disposable } from "./surfaces/types.js";
import type { SessionController } from "./session.js";
import { startSignIn } from "./auth/globus-login.js";
import { isLevel, LEVEL_INFO, LEVELS } from "./disclosure.js";

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
    // A level change re-mounts the current surface. Surfaces therefore read
    // ctx.level once at mount instead of subscribing — one update mechanism, and
    // it reuses route()'s existing dispose/mount path.
    //
    // The control is re-rendered too, not just the surface: a surface can raise the
    // level itself (guided mode's "I know what I need"), and without this the
    // header would still read "Guided" while the user was looking at the standard
    // view — the control lying about the one piece of state it exists to show.
    this.session.onLevelChange(() => {
      this.renderLevel();
      void this.route({ remount: true });
    });
    void this.route();
  }

  private render(): void {
    this.root.innerHTML = `
      <header class="portal-header">
        <a class="portal-brand" href="/" aria-label="spore.host home">
          <!-- A 3x-DPR derivative of spore-host-logo.png, not the master: the header
               renders it 26px tall, and shipping the 1983x793 original there would
               cost 914 KB on every page load for 65 CSS pixels of wordmark. -->
          <img class="portal-brand-img" src="../assets/brand/spore-host-logo-header.png"
               alt="spore.host" width="195" height="78">
        </a>
        <div class="portal-level"></div>
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
    this.renderLevel();
    this.renderAccount();
  }

  /**
   * The one portal-wide disclosure control, in the header beside the account block.
   *
   * A single visible control is the whole state — no per-surface toggles, which
   * would leave the user unable to say what mode they were in. The blurb is
   * rendered as the option's title so the choice explains itself; "Guided" alone
   * doesn't tell a first-timer what they'd get.
   */
  private renderLevel(): void {
    const el = this.root.querySelector<HTMLElement>(".portal-level")!;
    const current = this.session.level;
    const opts = LEVELS.map(
      (l) =>
        `<option value="${l}"${l === current ? " selected" : ""} title="${escapeHtml(
          LEVEL_INFO[l].blurb,
        )}">${escapeHtml(LEVEL_INFO[l].label)}</option>`,
    ).join("");
    el.innerHTML = `
      <label class="portal-level-label" for="portal-level-select">Detail</label>
      <select class="portal-level-select" id="portal-level-select"
              title="${escapeHtml(LEVEL_INFO[current].blurb)}">${opts}</select>`;
    const select = el.querySelector<HTMLSelectElement>(".portal-level-select")!;
    select.addEventListener("change", () => {
      // Guard rather than cast: a stale bookmarked value or a hand-edited DOM
      // must not put an unknown string into the level, where every atLeast()
      // comparison would silently read as below-guided.
      if (!isLevel(select.value)) return;
      this.session.setLevel(select.value);
      select.title = LEVEL_INFO[select.value].blurb;
    });
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
        // Dispose the mounted surface BEFORE clearing the session, and do it here
        // rather than leaving it to route().
        //
        // route() only disposes when the route id changes, and signing out doesn't
        // change it: location.hash = "" falls through currentId() to surfaces[0],
        // which is `instances` — the surface most likely to be mounted. So without
        // this, `this.current.id === id` and the dispose branch is skipped.
        //
        // That is not cosmetic. The instances surface holds a SpawnClient whose
        // startMonitor() interval is still running, over an EC2Provider that
        // captured the credential *values* — which session.clear() cannot
        // invalidate. The result is authenticated DescribeInstances calls
        // continuing after the user believes they signed out, until STS expiry.
        // The same applies to a live SSM session on the terminal surface.
        this.current?.disposable.dispose();
        this.current = null;
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
    // Strip a trailing ?query before splitting the path — a surface can be
    // returned to with params (e.g. the Slack OAuth callback lands on
    // #/slack?bot=connected), and the route id must still resolve.
    const id = location.hash.replace(/^#\/?/, "").split("?")[0]!.split("/")[0];
    return id || surfaces[0]?.id || "";
  }

  /**
   * Mount the surface for the current hash.
   *
   * `remount` forces a fresh mount of the *same* surface — needed when something
   * the surface read at mount time (the disclosure level) has changed. Without it
   * the `already mounted` short-circuit below would swallow the update.
   */
  private async route(opts: { remount?: boolean } = {}): Promise<void> {
    const id = this.currentId();
    const surface = findSurface(id);
    // Dispose the outgoing surface first.
    if (this.current && (this.current.id !== id || opts.remount)) {
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
      level: this.session.level,
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

  /**
   * The expiry banner.
   *
   * Both strings name the instances, because expiry stops the *portal* and not the
   * machines. The instances surface calls client.stopMonitor() when creds expire, so
   * the list freezes and the cost meter stops advancing while the instances keep
   * billing. "Session expired. Sign in again" is, at that moment, read as "nothing
   * is happening" — the opposite of what's true, and expensive to believe.
   */
  private showExpiry(state: "warning" | "expired"): void {
    this.bannerEl.hidden = false;
    if (state === "warning") {
      this.bannerEl.className = "portal-banner warn";
      this.bannerEl.textContent =
        "Your session expires soon — sign in again to stay connected. Anything you have running keeps running.";
    } else {
      this.bannerEl.className = "portal-banner expired";
      this.bannerEl.innerHTML = `Session expired — this page has stopped updating, but any
        machines you started are still running and still costing money.
        <button class="portal-reauth">Sign in again</button> to see and stop them.`;
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
