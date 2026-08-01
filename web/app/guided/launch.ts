// The guided launch panel: the picker, then one confirmation showing what it is,
// what it costs, and when it shuts itself off. Then a single button.
//
// It launches through the SAME SpawnClient the Dashboard is driven by, so the
// resulting instance appears in the Dashboard's own list via its existing event
// subscription. No second list, no polling of our own — the composition is the
// point, and duplicating the list is how the two views drift apart.

import type { SpawnClient } from "@spore-host/spawn-ts";
import { mountGuidedPicker, type GuidedChoice } from "./picker.js";

export interface GuidedLaunchOptions {
  client: SpawnClient;
  /** The region, shown in the confirmation — a cost figure without a region is half a fact. */
  region: string;
  /** "I know what I need" — hand control back to the caller (raise the level). */
  onEscape(): void;
}

/** Render the guided launch flow into `host`. Returns a dispose fn. */
export function mountGuidedLaunch(host: HTMLElement, opts: GuidedLaunchOptions): () => void {
  const root = document.createElement("div");
  root.className = "guided-launch";
  host.appendChild(root);

  let disposeStep: (() => void) | null = null;
  let live = true;

  const showPicker = (): void => {
    disposeStep?.();
    root.innerHTML = "";
    disposeStep = mountGuidedPicker(root, {
      onChoose: (choice) => showConfirm(choice),
      onEscape: opts.onEscape,
    });
  };

  const showConfirm = (choice: GuidedChoice): void => {
    disposeStep?.();
    disposeStep = null;
    const { shape, rec } = choice;
    const i = rec.pick.instance;

    // The cost line is deliberately the total, and deliberately says "up to":
    // the TTL is a ceiling, not a prediction — an instance that finishes early
    // and is stopped costs less, and promising the exact figure would be wrong in
    // the user's favour right up until it wasn't.
    const cost =
      rec.pricePerHour == null
        ? `<p class="guided-cost unknown">We don't have a price for
             <code>${escapeHtml(i.instanceType)}</code> in the offline catalog, so we
             can't tell you what this costs. Check the AWS pricing page before you
             launch.</p>`
        : `<p class="guided-cost">Up to <b>$${rec.estimatedTotal!.toFixed(2)}</b>
             — $${rec.pricePerHour.toFixed(4)} per hour${
               rec.priceIsEstimate ? " (estimated)" : ""
             }, for at most ${shape.defaultTtlHours} hours.</p>`;

    root.innerHTML = `
      <div class="guided-confirm">
        <h2>${escapeHtml(shape.label)}</h2>
        <dl class="guided-facts">
          <dt>Machine</dt><dd><code>${escapeHtml(i.instanceType)}</code> —
            ${i.vcpus} vCPU, ${(i.memoryMib / 1024).toFixed(0)} GiB${
              i.gpus ? `, ${i.gpus}× ${escapeHtml(i.gpuModel ?? "GPU")}` : ""
            }</dd>
          <dt>Region</dt><dd>${escapeHtml(opts.region)}</dd>
          <dt>Shuts down</dt><dd>automatically after
            ${shape.defaultTtlHours} hours, whatever happens</dd>
        </dl>
        ${cost}
        <p class="guided-reassure">The time limit is enforced on the machine itself, so
          it applies even if you close this tab.</p>
        <div class="guided-actions">
          <button class="guided-back" type="button">← Something else</button>
          <button class="guided-go" type="button">Start it</button>
        </div>
        <div class="guided-msg" aria-live="polite"></div>
      </div>`;

    const back = root.querySelector<HTMLButtonElement>(".guided-back")!;
    const go = root.querySelector<HTMLButtonElement>(".guided-go")!;
    const msg = root.querySelector<HTMLElement>(".guided-msg")!;

    back.addEventListener("click", () => showPicker());
    go.addEventListener("click", () => {
      go.disabled = true;
      back.disabled = true;
      msg.className = "guided-msg";
      msg.textContent = "Starting…";
      void opts.client
        .launch({
          name: instanceName(shape.id),
          instanceType: i.instanceType,
          region: opts.region,
          spot: false, // A beginner's first instance should not vanish mid-run.
          ttl: `${shape.defaultTtlHours}h`,
          // Pass the price through so the Dashboard's cost meter reflects THIS
          // instance rather than its hardcoded default. Zero when unknown is
          // correct here: it disables the meter's math instead of inventing a rate.
          pricePerHour: rec.pricePerHour ?? 0,
        })
        .then((inst) => {
          if (!live) return;
          msg.className = "guided-msg ok";
          msg.textContent = `Started ${inst.name} (${inst.instanceId}). It's in the list below.`;
        })
        .catch((err: unknown) => {
          if (!live) return;
          // Show the real error. A guided user is the least able to diagnose a
          // vague one, so "couldn't start it" alone is the least helpful thing we
          // could say — quota and permission failures both land here and read
          // completely differently.
          msg.className = "guided-msg error";
          msg.textContent = `Couldn't start it: ${(err as Error).message}`;
          go.disabled = false;
          back.disabled = false;
        });
    });
  };

  showPicker();

  return () => {
    live = false;
    disposeStep?.();
    root.remove();
  };
}

/**
 * A name the user can recognise in the list later.
 *
 * Suffixed with a timestamp because a guided user will click the same shape twice
 * and two instances called "gpu" are indistinguishable in the list — which is
 * exactly the moment they need to tell them apart to terminate the right one.
 */
function instanceName(shapeId: string): string {
  const stamp = new Date().toISOString().slice(5, 16).replace(/[-:T]/g, "");
  return `${shapeId}-${stamp}`;
}

function escapeHtml(s: string): string {
  return s.replace(
    /[&<>"']/g,
    (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" })[c]!,
  );
}
