// The spawn surface: launch / list / terminate EC2 in the signed-in account,
// entirely browser-native. Mounts the prebuilt Dashboard from
// @spore-host/spawn-ts/ui (launch form, live cards, meters, orphans, sweeps,
// embedded truffle picker), driven by a SpawnClient over EC2Provider.
import { SpawnClient } from "@spore-host/spawn-ts";
import { EC2Provider } from "@spore-host/spawn-ts";
import { Dashboard, confirmDialog } from "@spore-host/spawn-ts/ui";
import type { Disposable, SurfaceContext, ToolSurface } from "./types.js";

export const instancesSurface: ToolSurface = {
  id: "instances",
  label: "Instances",
  accent: "--spawn",
  requiresAuth: true,

  async mount(host: HTMLElement, ctx: SurfaceContext): Promise<Disposable> {
    const creds = ctx.session.getCreds();
    if (!creds) throw new Error("instances surface mounted without a session");

    const client = new SpawnClient({
      provider: new EC2Provider({
        region: ctx.session.region,
        accessKeyId: creds.accessKeyId,
        secretAccessKey: creds.secretAccessKey,
        sessionToken: creds.sessionToken,
        // Required for launched instances to self-terminate and call the infra
        // DNS Lambda (spored reads spawn:* tags via this profile).
        iamInstanceProfile: "spored-instance-profile",
      }),
    });
    client.startMonitor();

    const dashboard = new Dashboard(client, confirmDialog);
    host.appendChild(dashboard.el);

    // Tear the client's polling down if the session's creds expire.
    const offExpiry = ctx.session.onExpiry((state) => {
      if (state === "expired") client.stopMonitor();
    });

    return {
      dispose() {
        offExpiry();
        client.stopMonitor();
        dashboard.el.remove();
      },
    };
  },
};
