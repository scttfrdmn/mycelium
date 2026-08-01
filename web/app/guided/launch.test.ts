// The guided launch panel, driven against a stub SpawnClient.
//
// The real path needs federated AWS credentials and would bill for an instance, so
// launch() is stubbed and what's asserted is the *shape of the request* — which is
// where the cost-safety properties live. A guided launch with no TTL, or on spot,
// would be a real defect that no amount of live clicking would reveal until it
// cost someone money.
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { SpawnClient } from "@spore-host/spawn-ts";
import { mountGuidedLaunch } from "./launch.js";
import { GUIDED_SHAPES } from "./shapes.js";

const settle = () => new Promise((r) => setTimeout(r, 0));

/** A SpawnClient stand-in exposing only what the panel calls. */
function stubClient(impl?: (input: unknown) => Promise<unknown>) {
  const launch = vi.fn(
    impl ?? (async (input: any) => ({ instanceId: "i-0abc", name: input.name })),
  );
  return { client: { launch } as unknown as SpawnClient, launch };
}

describe("mountGuidedLaunch", () => {
  let host: HTMLElement;

  beforeEach(() => {
    document.body.innerHTML = "";
    host = document.createElement("div");
    document.body.appendChild(host);
  });

  async function pickFirst(client: SpawnClient, onEscape = vi.fn()) {
    const dispose = mountGuidedLaunch(host, { client, region: "us-east-1", onEscape });
    await settle();
    host.querySelector<HTMLButtonElement>(".guided-card")!.click();
    await settle();
    return dispose;
  }

  it("shows the picker first", async () => {
    const { client } = stubClient();
    const dispose = mountGuidedLaunch(host, { client, region: "us-east-1", onEscape: vi.fn() });
    await settle();
    expect(host.querySelectorAll(".guided-card")).toHaveLength(GUIDED_SHAPES.length);
    expect(host.querySelector(".guided-confirm")).toBeNull();
    dispose();
  });

  it("confirms with the machine, region, cost and shutdown before launching", async () => {
    const { client, launch } = stubClient();
    const dispose = await pickFirst(client);

    const confirm = host.querySelector<HTMLElement>(".guided-confirm")!;
    expect(confirm).toBeTruthy();
    const text = confirm.textContent!.replace(/\s+/g, " ");
    expect(text).toMatch(/t4g|[a-z0-9-]+\.[a-z0-9]+/); // a real instance type
    expect(text).toContain("us-east-1");
    expect(text).toMatch(/\$\d/);
    // The shutdown promise is the one thing a guided user must see before they
    // commit — it's what makes an unattended instance safe.
    expect(text).toMatch(/hours/);
    expect(text.toLowerCase()).toContain("automatically");
    // Nothing launched yet: confirmation is a step, not a formality.
    expect(launch).not.toHaveBeenCalled();
    dispose();
  });

  it("says the limit is enforced on the machine, not by the tab", async () => {
    // A user who closes the tab must not be left thinking their instance now runs
    // forever — nor thinking it was killed. Both are wrong.
    const { client } = stubClient();
    const dispose = await pickFirst(client);
    expect(host.querySelector(".guided-reassure")!.textContent).toContain("close this tab");
    dispose();
  });

  it("launches with a TTL and without spot", async () => {
    const { client, launch } = stubClient();
    const dispose = await pickFirst(client);
    host.querySelector<HTMLButtonElement>(".guided-go")!.click();
    await settle();

    expect(launch).toHaveBeenCalledTimes(1);
    const input = launch.mock.calls[0]![0] as any;
    const shape = GUIDED_SHAPES[0]!;
    // A TTL is the hard cost backstop; a guided launch must never be unbounded.
    expect(input.ttl).toBe(`${shape.defaultTtlHours}h`);
    // Not spot: a beginner's first instance should not vanish mid-run with no
    // explanation they could act on.
    expect(input.spot).toBe(false);
    expect(input.region).toBe("us-east-1");
    expect(input.instanceType).toMatch(/^[a-z0-9-]+\.[a-z0-9]+$/);
    // The price travels with the launch so the Dashboard's cost meter reflects
    // THIS instance instead of its hardcoded default.
    expect(input.pricePerHour).toBeGreaterThan(0);
    dispose();
  });

  it("names the instance distinguishably", async () => {
    // A guided user will click the same shape twice, and two instances with the
    // same name are indistinguishable at exactly the moment they need to terminate
    // one of them.
    const { client, launch } = stubClient();
    const d1 = await pickFirst(client);
    host.querySelector<HTMLButtonElement>(".guided-go")!.click();
    await settle();
    const name = (launch.mock.calls[0]![0] as any).name as string;
    expect(name).toContain(GUIDED_SHAPES[0]!.id);
    expect(name.length).toBeGreaterThan(GUIDED_SHAPES[0]!.id.length + 1);
    d1();
  });

  it("reports success with the instance id", async () => {
    const { client } = stubClient();
    const dispose = await pickFirst(client);
    host.querySelector<HTMLButtonElement>(".guided-go")!.click();
    await settle();
    const msg = host.querySelector<HTMLElement>(".guided-msg")!;
    expect(msg.className).toContain("ok");
    expect(msg.textContent).toContain("i-0abc");
    dispose();
  });

  it("surfaces the real error text on failure and re-enables the button", async () => {
    // A guided user is the least able to diagnose a vague error, so "couldn't
    // start it" alone is the least helpful thing we could say — a quota failure
    // and a permission failure both land here and read completely differently.
    const { client } = stubClient(async () => {
      throw new Error("VcpuLimitExceeded: you have requested more vCPU capacity");
    });
    const dispose = await pickFirst(client);
    const go = host.querySelector<HTMLButtonElement>(".guided-go")!;
    go.click();
    await settle();

    const msg = host.querySelector<HTMLElement>(".guided-msg")!;
    expect(msg.className).toContain("error");
    expect(msg.textContent).toContain("VcpuLimitExceeded");
    // Retryable: a quota bump or a different shape may well work.
    expect(go.disabled).toBe(false);
    dispose();
  });

  it("disables the button while launching so one click is one instance", async () => {
    let release!: () => void;
    const { client, launch } = stubClient(
      () => new Promise((r) => (release = () => r({ instanceId: "i-1", name: "n" }))),
    );
    const dispose = await pickFirst(client);
    const go = host.querySelector<HTMLButtonElement>(".guided-go")!;
    go.click();
    await settle();
    expect(go.disabled).toBe(true);
    go.click(); // an impatient second click must not launch a second instance
    go.click();
    await settle();
    expect(launch).toHaveBeenCalledTimes(1);
    release();
    await settle();
    dispose();
  });

  it("lets the user go back to the picker without launching", async () => {
    const { client, launch } = stubClient();
    const dispose = await pickFirst(client);
    host.querySelector<HTMLButtonElement>(".guided-back")!.click();
    await settle();
    expect(host.querySelectorAll(".guided-card")).toHaveLength(GUIDED_SHAPES.length);
    expect(host.querySelector(".guided-confirm")).toBeNull();
    expect(launch).not.toHaveBeenCalled();
    dispose();
  });

  it("passes the escape hatch through", async () => {
    const onEscape = vi.fn();
    const { client } = stubClient();
    const dispose = mountGuidedLaunch(host, { client, region: "us-east-1", onEscape });
    await settle();
    host.querySelector<HTMLButtonElement>(".guided-escape")!.click();
    expect(onEscape).toHaveBeenCalledTimes(1);
    dispose();
  });

  it("does not write a result into a detached DOM after dispose", async () => {
    let release!: () => void;
    const { client } = stubClient(
      () => new Promise((r) => (release = () => r({ instanceId: "i-2", name: "n" }))),
    );
    const dispose = await pickFirst(client);
    host.querySelector<HTMLButtonElement>(".guided-go")!.click();
    await settle();
    dispose();
    release();
    await settle();
    expect(host.querySelector(".guided-launch")).toBeNull();
  });
});
