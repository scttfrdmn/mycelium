// The terminal surface: a live shell into a portal-launched instance over AWS
// SSM Session Manager — no SSH, no port 22, no key. The browser calls
// ssm:StartSession with the session's in-memory creds, gets a session-scoped
// StreamUrl + token, and attachTerminal (from @spore-host/spawn-ts/terminal)
// opens the data channel and renders with xterm. On teardown we close the socket
// AND TerminateSession (StartSession leaks a server-side session otherwise).
import { SSMClient, StartSessionCommand, TerminateSessionCommand } from "@aws-sdk/client-ssm";
import { attachTerminal, type AttachedTerminal } from "@spore-host/spawn-ts/terminal";
import type { Disposable, SurfaceContext, ToolSurface } from "./types.js";

export const terminalSurface: ToolSurface = {
  id: "terminal",
  label: "Terminal",
  accent: "--spored",
  requiresAuth: true,

  async mount(host: HTMLElement, ctx: SurfaceContext): Promise<Disposable> {
    const creds = ctx.session.getCreds();
    if (!creds) throw new Error("terminal surface mounted without a session");
    const region = ctx.session.region;
    const credentials = {
      accessKeyId: creds.accessKeyId,
      secretAccessKey: creds.secretAccessKey,
      sessionToken: creds.sessionToken,
    };

    const root = document.createElement("div");
    root.className = "terminal-surface";
    root.innerHTML = `
      <div class="terminal-connect">
        <h2>Connect a terminal</h2>
        <p class="terminal-hint">Opens a shell over SSM Session Manager (no SSH). The
          instance must have the SSM agent + an instance profile allowing SSM —
          spawn-launched instances do.</p>
        <form class="terminal-form">
          <input class="terminal-target" type="text" placeholder="i-0123456789abcdef0" autocomplete="off" spellcheck="false" />
          <button type="submit" class="terminal-open">Connect</button>
          <button type="button" class="terminal-close" hidden>Disconnect</button>
        </form>
        <div class="terminal-status" aria-live="polite"></div>
      </div>
      <div class="terminal-host" hidden></div>`;
    host.appendChild(root);

    const form = root.querySelector<HTMLFormElement>(".terminal-form")!;
    const target = root.querySelector<HTMLInputElement>(".terminal-target")!;
    const openBtn = root.querySelector<HTMLButtonElement>(".terminal-open")!;
    const closeBtn = root.querySelector<HTMLButtonElement>(".terminal-close")!;
    const status = root.querySelector<HTMLElement>(".terminal-status")!;
    const termHost = root.querySelector<HTMLElement>(".terminal-host")!;

    let attached: AttachedTerminal | null = null;
    let sessionId: string | null = null;

    // Best-effort SSM cleanup: close the socket + TerminateSession (a bare
    // StartSession leaves a live session server-side until it times out).
    async function teardown(): Promise<void> {
      attached?.dispose();
      attached = null;
      const sid = sessionId;
      sessionId = null;
      if (sid) {
        try {
          await new SSMClient({ region, credentials }).send(new TerminateSessionCommand({ SessionId: sid }));
        } catch {
          // The session may already be gone (agent exit); ignore.
        }
      }
    }

    function resetUi(): void {
      termHost.hidden = true;
      termHost.innerHTML = "";
      closeBtn.hidden = true;
      openBtn.disabled = false;
      target.disabled = false;
    }

    async function connect(instanceId: string): Promise<void> {
      const id = instanceId.trim();
      if (!/^i-[0-9a-f]{8,}$/.test(id)) {
        status.textContent = "enter a valid instance id (i-…)";
        return;
      }
      openBtn.disabled = true;
      target.disabled = true;
      status.textContent = "starting SSM session…";
      try {
        const ssm = new SSMClient({ region, credentials });
        const started = await ssm.send(new StartSessionCommand({ Target: id }));
        if (!started.StreamUrl || !started.TokenValue || !started.SessionId) {
          throw new Error("StartSession returned an incomplete session");
        }
        sessionId = started.SessionId;
        status.textContent = "";
        // Un-hide BEFORE attach so xterm's fit addon measures a real size.
        termHost.hidden = false;
        attached = await attachTerminal(
          termHost,
          { streamUrl: started.StreamUrl, tokenValue: started.TokenValue, sessionId: started.SessionId },
          (reason) => {
            status.textContent = `session closed${reason ? ": " + reason : ""}`;
            void teardown();
            resetUi();
          },
        );
        closeBtn.hidden = false;
      } catch (err) {
        status.textContent = `couldn't connect: ${(err as Error).message}`;
        await teardown();
        resetUi();
      }
    }

    const onSubmit = (e: Event) => {
      e.preventDefault();
      void connect(target.value);
    };
    const onClose = () => {
      void teardown();
      resetUi();
      status.textContent = "disconnected";
    };
    form.addEventListener("submit", onSubmit);
    closeBtn.addEventListener("click", onClose);

    // If the session's creds expire, drop the live shell.
    const offExpiry = ctx.session.onExpiry((state) => {
      if (state === "expired") {
        void teardown();
        resetUi();
        status.textContent = "session expired — sign in again";
      }
    });

    return {
      dispose() {
        offExpiry();
        form.removeEventListener("submit", onSubmit);
        closeBtn.removeEventListener("click", onClose);
        void teardown();
        root.remove();
      },
    };
  },
};
