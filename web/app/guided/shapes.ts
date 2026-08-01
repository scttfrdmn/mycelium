// The curated picker — guided mode's answer to "what are you doing?" when there
// is no AI available.
//
// This is the load-bearing piece of progressive disclosure, and the dependency
// direction is the point: **the simplest mode has the fewest dependencies.** It
// needs truffle-ts's offline catalog and nothing else — no Bedrock, no model
// access, no credentials, not even a network. advisor-ts, when present, replaces
// the fixed list with a free-text question in the same slot; it does not become a
// prerequisite for the beginner's entry point.
//
// The wrong way round would be to make guided mode the AI mode, because then the
// path a first-time user takes is the one most likely to be broken — and it's the
// one nobody developing the portal is ever in, so nobody would notice.
//
// Every shape here is a real truffle-ts query whose results were checked, not a
// plausible-looking string. A query that returns nothing renders as an honest
// "couldn't resolve this" rather than an empty card.

import { find, type FindResult } from "@spore-host/truffle-ts";

export interface GuidedShape {
  id: string;
  /** What the user recognises about their own work — not hardware vocabulary. */
  label: string;
  /** One line on what this is for, in the same register as the label. */
  blurb: string;
  /**
   * The truffle-ts query. Phrased as MINIMUMS (truffle treats vcpus/memory as
   * ">=") and then price-ranked, so the user gets the cheapest machine that
   * clears the bar rather than the largest one that matches.
   */
  query: string;
  /**
   * A sensible time limit in hours, pre-filled so the beginner's launch has a
   * cost ceiling without them having to think about TTL. Every guided launch gets
   * one: an instance with no TTL is the single most expensive mistake available
   * here, and "I forgot it was running" is the normal way it happens.
   */
  defaultTtlHours: number;
  /**
   * Whether this shape is actually asking for an accelerator.
   *
   * When false, GPU instances are excluded from the pick even if one is cheapest.
   * This is not a preference — it's a correctness rule. `find("8 vcpus 128gb")`
   * returns `g3.4xlarge` (16 vCPU, 128 GiB, 1×M60) as the cheapest priced match,
   * so a user who asked for memory would be handed a decade-old GPU box and
   * charged for an accelerator they never wanted and cannot use. Filtering to
   * CPU-only yields `r7g.4xlarge` — the right answer to the question asked.
   */
  wantsGpu: boolean;
}

/**
 * The curated list. Ordered cheapest-intent first, because the list is read top
 * to bottom and the first plausible match tends to get picked.
 *
 * "I know what I need" is deliberately the last entry and is not a shape — it's
 * the exit to standard mode. Guided mode must not be a trap.
 */
export const GUIDED_SHAPES: readonly GuidedShape[] = [
  {
    id: "small-analysis",
    label: "A small analysis",
    blurb: "Notebooks, scripts, modest data. The usual starting point.",
    query: "4 vcpus 16gb",
    defaultTtlHours: 4,
    wantsGpu: false,
  },
  {
    id: "lots-of-memory",
    label: "Something that needs a lot of memory",
    blurb: "A big table or genome that has to fit in RAM at once.",
    query: "8 vcpus 128gb",
    defaultTtlHours: 4,
    wantsGpu: false,
  },
  {
    id: "lots-of-cores",
    label: "Lots of CPU cores",
    blurb: "Work that splits across many cores on one machine.",
    query: "64 vcpus",
    defaultTtlHours: 4,
    wantsGpu: false,
  },
  {
    id: "gpu",
    label: "Something with a GPU",
    blurb: "Training, inference, or anything CUDA. Starts with a smaller GPU.",
    // Named part rather than bare "gpu": bare "gpu" parses to an unknown token
    // and "80gb" filters system RAM rather than VRAM (truffle-ts#37). An L4 is
    // the right first GPU — real, current, and not $55/hr.
    query: "nvidia l4",
    defaultTtlHours: 2,
    wantsGpu: true,
  },
  {
    id: "big-gpu",
    label: "A big GPU for training",
    blurb: "H100-class. Expensive — check the hourly cost before you launch.",
    query: "nvidia h100",
    // Two hours, not four: at H100 prices the TTL is the cost control, and a
    // forgotten instance here costs more per hour than most of the others cost
    // per day.
    defaultTtlHours: 2,
    wantsGpu: true,
  },
] as const;

/** A resolved recommendation: the shape, the machine, and what it costs. */
export interface GuidedRecommendation {
  shape: GuidedShape;
  /** The cheapest catalog match that clears the shape's minimums. */
  pick: FindResult;
  /** $/hr for `pick`, or undefined when the catalog has no usable price. */
  pricePerHour?: number;
  /** pricePerHour × defaultTtlHours, when a price is known. */
  estimatedTotal?: number;
  /**
   * True when the catalog flagged this price as an estimate rather than a pulled
   * figure. Surfaced so the UI can mark it — an estimate presented as a price is
   * the same defect as a fabricated one, just smaller.
   */
  priceIsEstimate: boolean;
  /**
   * How many types matched before we picked one. Shown so "we chose this" doesn't
   * read as "this is the only option".
   */
  totalMatches: number;
}

/**
 * Resolve one shape to a concrete recommendation.
 *
 * Returns undefined when the query matches nothing, so the caller can say so
 * rather than render an empty card. Does NOT catch errors — a broken catalog is a
 * failure and must not look like "no matches", which is a fact about the catalog's
 * contents.
 */
export async function resolveShape(
  shape: GuidedShape,
  finder: typeof find = find,
): Promise<GuidedRecommendation | undefined> {
  const found = await finder(shape.query);
  if (found.length === 0) return undefined;

  const pick = cheapest(found, shape.wantsGpu) ?? found[0]!;
  const price = usablePrice(pick);

  return {
    shape,
    pick,
    pricePerHour: price,
    estimatedTotal: price != null ? price * shape.defaultTtlHours : undefined,
    priceIsEstimate: pick.instance.estimatedPrice === true,
    totalMatches: found.length,
  };
}

/** Resolve every shape, preserving list order. */
export async function resolveAllShapes(
  finder: typeof find = find,
  shapes: readonly GuidedShape[] = GUIDED_SHAPES,
): Promise<Array<GuidedRecommendation | undefined>> {
  return Promise.all(shapes.map((s) => resolveShape(s, finder)));
}

/**
 * A price we're willing to show, or undefined.
 *
 * Rejects zero as well as null, and both halves still earn their keep:
 *
 * - **Null** is the normal case for a type with no on-demand row at all — since
 *   truffle-ts 0.5.0, `p6e-gb200.36xlarge` and `p5e.48xlarge` carry no price
 *   rather than a guessed one, so a price-ranked picker must skip them instead of
 *   ranking them at 0.
 * - **Zero** is defence in depth. truffle-ts now refuses to write a non-positive
 *   price (truffle-ts#42), so this shouldn't fire — but a zero is the single most
 *   damaging wrong price here, because it *sorts first*: a naive "cheapest option"
 *   picker recommends an H100 box, at no charge, ahead of a $0.13 t4g. That's a
 *   one-line guard against an upstream regression that no amount of clicking would
 *   reveal, so it stays.
 *
 * This replaced a named quarantine of `p6e-gb200.36xlarge` (fabricated at
 * $0.2000/hr for 72×B200) and `p5.4xlarge` (`onDemandPrice: 0`), removed once
 * truffle-ts#39 and #42 landed in 0.5.0. Worth recording *why* that list was
 * names rather than a plausibility threshold: no per-vCPU floor could separate
 * those entries from real cheap instances — `t4g.nano` is legitimately
 * $0.0021/vCPU against the bad B200's $0.00139, only 34% apart — so any threshold
 * strict enough to catch the fabrication also rejected honest bargains. If a
 * future bad price shows up, name it again; don't invent a threshold.
 */
function usablePrice(r: FindResult): number | undefined {
  const p = r.instance.onDemandPrice;
  return typeof p === "number" && p > 0 ? p : undefined;
}

/**
 * The cheapest match with a usable price, honouring the shape's GPU intent.
 *
 * This re-sort is necessary, not redundant. truffle-ts treats `vcpus`/`memory` as
 * MINIMUMS and ranks by its own size preference, so `find("4 vcpus 16gb")`
 * returns `r8g.12xlarge` (48 vCPU, 384 GiB, **$2.83/hr**) first out of 194
 * matches — while `t4g.xlarge` (exactly 4 vCPU / 16 GiB, **$0.13/hr**) is further
 * down. Handing the user truffle's first result would quote them 21× the cost of
 * the right answer for the thing they asked for.
 *
 * Returns undefined when nothing survives the filters, so the caller falls back to
 * the top match and reports the price as unknown rather than inventing one.
 */
function cheapest(found: FindResult[], wantsGpu: boolean): FindResult | undefined {
  let candidates = found.filter((r) => usablePrice(r) != null);

  if (!wantsGpu) {
    const cpuOnly = candidates.filter((r) => !r.instance.gpus);
    // Only narrow if something remains: a shape that *only* matches GPU boxes
    // should still get an answer, and "no recommendation" would be a worse
    // outcome than an unexpected accelerator we can at least show the price of.
    if (cpuOnly.length > 0) candidates = cpuOnly;
  }

  let best: FindResult | undefined;
  let bestPrice = Infinity;
  for (const r of candidates) {
    const p = usablePrice(r)!;
    if (p < bestPrice) {
      best = r;
      bestPrice = p;
    }
  }
  return best;
}
