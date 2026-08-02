// The guided picker's rendering: "what are you doing?" as a short list of
// recognisable shapes, each resolved to a real instance type with a real price.
//
// Kept separate from shapes.ts so the selection logic (which is the part with
// correctness rules and upstream data caveats) stays free of DOM code.
//
// This renders in the same slot advisor-ts will eventually occupy with a free-text
// question. The slot is the same; the dependency is not — this needs only the
// bundled catalog, so it works with no credentials, no network, and no model
// access. That's the right way round: the beginner's entry point must not be the
// one that breaks first.

import { find } from "@spore-host/truffle-ts";
import {
  GUIDED_SHAPES,
  resolveShape,
  type GuidedRecommendation,
  type GuidedShape,
} from "./shapes.js";

/** What the picker reports when the user commits to a shape. */
export interface GuidedChoice {
  shape: GuidedShape;
  rec: GuidedRecommendation;
}

export interface GuidedPickerOptions {
  /** Called when the user picks a shape and clicks through. */
  onChoose(choice: GuidedChoice): void;
  /**
   * Called for "I know what I need" — the exit to a denser mode. Guided mode must
   * not be a trap: a user who outgrows it mid-task needs a way out that isn't
   * "hunt for the setting in the header".
   */
  onEscape(): void;
  /**
   * Catalog lookup, injected. Defaults to truffle-ts's `find`.
   *
   * Present so the no-match and lookup-failed branches are testable — they are
   * the two states most likely to ship broken, because the real catalog never
   * produces either.
   */
  finder?: typeof find;
  /** Shapes to offer. Defaults to the curated list; overridable for tests. */
  shapes?: readonly GuidedShape[];
}

/**
 * Render the picker into `host`. Returns a dispose fn.
 *
 * Each card resolves independently and renders as soon as it can, rather than
 * awaiting all five: a `find()` over the bundled catalog is fast, but one slow or
 * failed shape must not hold the whole list blank.
 */
export function mountGuidedPicker(host: HTMLElement, opts: GuidedPickerOptions): () => void {
  const root = document.createElement("div");
  root.className = "guided-picker";
  root.innerHTML = `
    <h2>What are you doing?</h2>
    <p class="guided-hint">Pick the closest match. We'll choose a machine, show what
      it costs per hour, and shut it down for you when the time's up.</p>
    <div class="guided-cards"></div>
    <button class="guided-escape" type="button">I know what I need →</button>`;
  host.appendChild(root);

  const cards = root.querySelector<HTMLElement>(".guided-cards")!;
  const escape = root.querySelector<HTMLButtonElement>(".guided-escape")!;
  const onEscape = () => opts.onEscape();
  escape.addEventListener("click", onEscape);

  // Track so dispose() can't leave a resolved promise writing into a detached DOM.
  let live = true;

  for (const shape of opts.shapes ?? GUIDED_SHAPES) {
    const card = document.createElement("button");
    card.type = "button";
    card.className = "guided-card loading";
    card.disabled = true;
    card.innerHTML = `
      <span class="guided-card-label">${escapeHtml(shape.label)}</span>
      <span class="guided-card-blurb">${escapeHtml(shape.blurb)}</span>
      <span class="guided-card-machine">finding a machine…</span>`;
    cards.appendChild(card);

    void resolveShape(shape, opts.finder ?? find)
      .then((rec) => {
        if (!live) return;
        card.classList.remove("loading");
        if (!rec) {
          // Say which fact is missing. "Unavailable" alone would leave the user
          // unable to tell a catalog gap from a broken portal.
          card.classList.add("unresolved");
          card.querySelector(".guided-card-machine")!.textContent =
            "no machine in the catalog matches this — try another option";
          return;
        }
        card.classList.add("ready");
        card.disabled = false;
        card.querySelector(".guided-card-machine")!.innerHTML = describe(rec);
        card.addEventListener("click", () => opts.onChoose({ shape, rec }));
      })
      .catch((err: unknown) => {
        if (!live) return;
        // A thrown find() is a failure, and it must not render as "no match" —
        // that would report a fact about the catalog's contents that we don't have.
        card.classList.remove("loading");
        card.classList.add("errored");
        card.querySelector(".guided-card-machine")!.textContent =
          `couldn't look this up: ${(err as Error).message}`;
      });
  }

  return () => {
    live = false;
    escape.removeEventListener("click", onEscape);
    root.remove();
  };
}

/**
 * The one line of hardware the guided user sees.
 *
 * Leads with the total, not the hourly rate: "$0.54 for 4 hours" is the number a
 * person can decide against, whereas $0.1344/hr requires them to do arithmetic
 * about a duration they haven't been told about yet.
 *
 * An unknown price says so. Rendering "$0.00" or omitting the cost entirely would
 * both read as "free".
 *
 * The match count is rendered because `resolveShape` computes it for exactly this
 * purpose — "cheapest of 194 that fit" tells the user a choice was made on their
 * behalf and roughly how much was on the table, where the bare recommendation reads
 * as the only thing available.
 */
function describe(rec: GuidedRecommendation): string {
  const i = rec.pick.instance;
  const gib = (i.memoryMib / 1024).toFixed(i.memoryMib % 1024 === 0 ? 0 : 1);
  const gpu = i.gpus ? ` · ${i.gpus}× ${escapeHtml(i.gpuModel ?? "GPU")}` : "";
  const specs = `${escapeHtml(i.instanceType)} — ${i.vcpus} vCPU · ${gib} GiB${gpu}`;
  const of =
    rec.totalMatches > 1
      ? `<span class="guided-card-alts">cheapest of ${rec.totalMatches} that fit</span>`
      : "";

  if (rec.pricePerHour == null) {
    return `<b>${specs}</b><span class="guided-card-cost unknown">price unknown for this
      type — check before launching</span>${of}`;
  }
  const est = rec.priceIsEstimate ? " (estimated)" : "";
  return `<b>${specs}</b><span class="guided-card-cost">about
    $${rec.estimatedTotal!.toFixed(2)} for ${rec.shape.defaultTtlHours} hours
    ($${rec.pricePerHour.toFixed(4)}/hr${est}, compute only)</span>${of}`;
}

function escapeHtml(s: string): string {
  return s.replace(
    /[&<>"']/g,
    (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" })[c]!,
  );
}
