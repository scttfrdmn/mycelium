// The shell's lifecycle contract with its surfaces: when it disposes them, and
// whether the level control tells the truth.
//
// The registry is mocked with a recording stub rather than the real surfaces. This
// is a test about the shell's dispose/mount discipline, and mounting the real
// surfaces would drag in the EC2/SSM SDKs and a live Dashboard to observe it.
import { beforeEach, describe, expect, it, vi } from "vitest";

const disposed: string[] = [];
const mounted: string[] = [];

vi.mock("./surfaces/registry.js", () => {
  const make = (id: string, requiresAuth: boolean) => ({
    id,
    label: id,
    accent: "--spawn",
    requiresAuth,
    async mount(host: HTMLElement) {
      mounted.push(id);
      host.innerHTML = `<div class="stub-${id}"></div>`;
      return {
        dispose() {
          disposed.push(id);
        },
      };
    },
  });
  const surfaces = [make("instances", true), make("truffle", false)];
  return { surfaces, findSurface: (id: string) => surfaces.find((s) => s.id === id) };
});

vi.mock("./auth/globus-login.js", () => ({ startSignIn: vi.fn() }));

const { Shell } = await import("./shell.js");
const { SessionController } = await import("./session.js");
const settle = () => new Promise((r) => setTimeout(r, 0));

/** A session that reports as signed in without touching STS. */
function signedInSession() {
  const session = new SessionController("us-east-1", null);
  vi.spyOn(session, "signedIn", "get").mockReturnValue(true);
  vi.spyOn(session, "accountId", "get").mockReturnValue("123456789012");
  vi.spyOn(session, "getCreds").mockReturnValue({
    accessKeyId: "AKIAIOSFODNN7EXAMPLE",
    secretAccessKey: "secret",
    sessionToken: "token",
  } as never);
  return session;
}

describe("Shell sign-out", () => {
  let root: HTMLElement;

  beforeEach(() => {
    disposed.length = 0;
    mounted.length = 0;
    location.hash = "";
    document.body.innerHTML = "";
    root = document.createElement("div");
    document.body.appendChild(root);
  });

  it("disposes the mounted surface on sign-out", async () => {
    // The bug this pins down: route() only disposes when the route id CHANGES, and
    // signing out doesn't change it — location.hash = "" falls through currentId()
    // to surfaces[0], which is `instances`. So `this.current.id === id` and the
    // dispose branch was skipped.
    //
    // That is not cosmetic. The instances surface holds a SpawnClient whose
    // startMonitor() interval is still running over an EC2Provider that captured the
    // credential VALUES — which session.clear() cannot invalidate. The user who signs
    // out keeps making authenticated DescribeInstances calls until STS expiry.
    const session = signedInSession();
    const shell = new Shell(root, session, { region: "us-east-1" } as never);
    shell.start();
    await settle();
    expect(mounted).toContain("instances");
    const disposeMark = disposed.length;

    // Sign out for real: the mock's signedIn getter has to follow, or the shell
    // would re-mount the same surface and the assertion would pass for the wrong
    // reason.
    vi.spyOn(session, "signedIn", "get").mockReturnValue(false);
    root.querySelector<HTMLButtonElement>(".portal-signout")!.click();
    await settle();

    expect(disposed.slice(disposeMark)).toEqual(["instances"]);
  });

  it("does not re-mount an auth-gated surface after signing out", async () => {
    const session = signedInSession();
    const shell = new Shell(root, session, { region: "us-east-1" } as never);
    shell.start();
    await settle();
    const mountMark = mounted.length;

    vi.spyOn(session, "signedIn", "get").mockReturnValue(false);
    root.querySelector<HTMLButtonElement>(".portal-signout")!.click();
    await settle();

    expect(mounted.slice(mountMark)).toEqual([]); // not mounted a second time
    expect(root.querySelector(".portal-gate")).not.toBeNull();
  });

  it("disposes before clearing the session, so teardown still has its creds", async () => {
    // Ordering matters for the surfaces whose dispose() makes an AWS call — the
    // terminal's TerminateSession is the live example. Disposing after clear() would
    // leave a session open server-side.
    //
    // Both events land in one ordered list, so this asserts the interleaving rather
    // than just that each happened.
    const order: string[] = [];
    const session = signedInSession();
    vi.spyOn(session, "clear").mockImplementation(() => order.push("clear"));
    disposed.length = 0;

    const shell = new Shell(root, session, { region: "us-east-1" } as never);
    shell.start();
    await settle();

    // Mirror dispose() calls into the same list. The stub's own dispose pushes to
    // `disposed`, so watch that array's growth via a proxy on the recorded surface.
    const origPush = disposed.push.bind(disposed);
    disposed.push = ((...args: string[]) => {
      order.push("dispose");
      return origPush(...args);
    }) as typeof disposed.push;

    root.querySelector<HTMLButtonElement>(".portal-signout")!.click();
    await settle();
    disposed.push = origPush;

    expect(order).toEqual(["dispose", "clear"]);
  });
});

describe("Shell level changes", () => {
  let root: HTMLElement;

  beforeEach(() => {
    disposed.length = 0;
    mounted.length = 0;
    location.hash = "#/truffle";
    document.body.innerHTML = "";
    root = document.createElement("div");
    document.body.appendChild(root);
  });

  it("re-mounts the current surface when the level changes", async () => {
    // Surfaces read ctx.level once at mount rather than subscribing, so the remount
    // IS the update mechanism. If it stopped happening, every surface would silently
    // keep rendering the level the user had when they arrived.
    //
    // Measured as a delta, not an absolute count: Shell.start() registers a
    // window-level hashchange listener and the class has no teardown (there is one
    // Shell for the app's lifetime), so shells from earlier tests in this file still
    // react to the beforeEach hash change. Counting from a mark keeps this test
    // about the level, not about listener bookkeeping.
    const session = new SessionController("us-east-1", null);
    const shell = new Shell(root, session, { region: "us-east-1" } as never);
    shell.start();
    await settle();
    const mountMark = mounted.length;
    const disposeMark = disposed.length;

    session.setLevel("expert");
    await settle();
    expect(disposed.slice(disposeMark)).toContain("truffle");
    expect(mounted.slice(mountMark)).toContain("truffle");
  });
});
