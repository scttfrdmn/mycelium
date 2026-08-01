// These tests run against truffle-ts's REAL bundled catalog, not a fixture.
//
// That's deliberate and it's the point of the file. The plan's headline check is
// "set the level to guided with the advisor absent and confirm the curated picker
// still returns real instance types and live prices" — a fixture would pass that
// check while the shipped portal quoted a fabricated $0.20/hr B200. The catalog is
// offline and bundled, so there is no network cost to using the real thing.
//
// The trade-off is that a catalog regeneration can turn these red. That is the
// intended behaviour: a shape whose query stops matching, or whose cheapest match
// becomes a $55/hr box, is a defect in the portal's most vulnerable path and
// should fail loudly rather than ship.

import { describe, expect, it } from "vitest";
import type { FindResult } from "@spore-host/truffle-ts";
import { GUIDED_SHAPES, resolveAllShapes, resolveShape } from "./shapes.js";

describe("GUIDED_SHAPES", () => {
  it("has unique ids", () => {
    const ids = GUIDED_SHAPES.map((s) => s.id);
    expect(new Set(ids).size).toBe(ids.length);
  });

  it("gives every shape a TTL", () => {
    // A guided launch with no time limit is the single most expensive mistake
    // available in this UI, and "I forgot it was running" is the normal way it
    // happens. There is no valid zero here.
    for (const s of GUIDED_SHAPES) {
      expect(s.defaultTtlHours, s.id).toBeGreaterThan(0);
    }
  });

  it("keeps GPU shapes on shorter TTLs than CPU shapes", () => {
    // At H100 prices the TTL *is* the cost control: a forgotten p5 costs more per
    // hour than most of the others cost per day.
    const gpuMax = Math.max(...GUIDED_SHAPES.filter((s) => s.wantsGpu).map((s) => s.defaultTtlHours));
    const cpuMin = Math.min(...GUIDED_SHAPES.filter((s) => !s.wantsGpu).map((s) => s.defaultTtlHours));
    expect(gpuMax).toBeLessThanOrEqual(cpuMin);
  });

  it("labels shapes in user vocabulary, not hardware vocabulary", () => {
    // The whole reason guided mode exists is that "gpu with 80gb for training" is
    // only writable by someone who already knows what they need. A label carrying
    // an instance type would reintroduce exactly that.
    for (const s of GUIDED_SHAPES) {
      expect(s.label, s.id).not.toMatch(/\b[a-z]\d[a-z]*\.\w+\b/);
      expect(s.blurb.length, s.id).toBeGreaterThan(0);
    }
  });
});

describe("resolveShape against the real bundled catalog", () => {
  it("resolves every curated shape to a real instance type", async () => {
    // THE headline test. Every shape must produce a machine — an unresolvable
    // entry in the beginner's only menu is a dead end for the user least equipped
    // to work around it.
    const recs = await resolveAllShapes();
    expect(recs).toHaveLength(GUIDED_SHAPES.length);
    recs.forEach((rec, i) => {
      const shape = GUIDED_SHAPES[i]!;
      expect(rec, `${shape.id} resolved to nothing — query "${shape.query}" matched no instance`).toBeDefined();
      expect(rec!.pick.instance.instanceType, shape.id).toMatch(/^[a-z0-9-]+\.[a-z0-9]+$/);
      expect(rec!.totalMatches, shape.id).toBeGreaterThan(0);
    });
  });

  it("quotes a usable price and a total for every shape", async () => {
    const recs = await resolveAllShapes();
    for (const rec of recs) {
      // A zero would sort first and read as free; a null would leave the guided
      // user with no cost information at all. Both are failures here.
      expect(rec!.pricePerHour, rec!.shape.id).toBeGreaterThan(0);
      expect(rec!.estimatedTotal, rec!.shape.id).toBeCloseTo(
        rec!.pricePerHour! * rec!.shape.defaultTtlHours,
        6,
      );
    }
  });

  it("picks the CHEAPEST match, not truffle's first", async () => {
    // truffle-ts treats vcpus/memory as MINIMUMS and ranks by its own size
    // preference: find("4 vcpus 16gb") returns r8g.12xlarge (48 vCPU, 384 GiB,
    // ~$2.83/hr) first, while t4g.xlarge (exactly 4/16, ~$0.13/hr) is further
    // down. Handing over truffle's first result would quote 21× the right cost.
    const small = GUIDED_SHAPES.find((s) => s.id === "small-analysis")!;
    const rec = (await resolveShape(small))!;

    const { find } = await import("@spore-host/truffle-ts");
    const raw = await find(small.query);
    const rawFirstPrice = raw[0]!.instance.onDemandPrice;

    expect(rec.pricePerHour).toBeLessThanOrEqual(rawFirstPrice ?? Infinity);
    // And it must beat every other priced, non-quarantined candidate.
    const better = raw.filter(
      (r) => usable(r) && r.instance.onDemandPrice! < rec.pricePerHour! && !isQuarantined(r),
    );
    expect(better.map((r) => r.instance.instanceType)).toEqual([]);
  });

  it("does not hand a non-GPU shape a GPU instance", async () => {
    // With the bad-price types quarantined, "8 vcpus 128gb" otherwise picks
    // g3.4xlarge (1× M60) — charging a user who asked for memory for a
    // decade-old accelerator they cannot use.
    const recs = await resolveAllShapes();
    for (const rec of recs) {
      if (rec!.shape.wantsGpu) continue;
      expect(rec!.pick.instance.gpus ?? 0, `${rec!.shape.id} → ${rec!.pick.instance.instanceType}`).toBe(0);
    }
  });

  it("gives GPU shapes an actual GPU", async () => {
    const recs = await resolveAllShapes();
    for (const rec of recs) {
      if (!rec!.shape.wantsGpu) continue;
      expect(rec!.pick.instance.gpus ?? 0, rec!.shape.id).toBeGreaterThan(0);
    }
  });

  it("excludes the known-bad catalog prices", async () => {
    // truffle-ts#39 (p6e-gb200.36xlarge at $0.2000/hr for 72× B200) and #42
    // (p5.4xlarge at onDemandPrice 0) both WIN a naive price sort. A price-ranked
    // picker is maximally exposed to a fabricated low price.
    const recs = await resolveAllShapes();
    const picked = recs.map((r) => r!.pick.instance.instanceType);
    expect(picked).not.toContain("p6e-gb200.36xlarge");
    expect(picked).not.toContain("p5.4xlarge");
  });

  it("keeps the cheap CPU shape genuinely cheap", async () => {
    // A regression fence, not a spec. If a catalog change makes the beginner's
    // first instance cost dollars per hour, that must fail here rather than
    // surprise a first-time user.
    const rec = (await resolveShape(GUIDED_SHAPES.find((s) => s.id === "small-analysis")!))!;
    expect(rec.pricePerHour).toBeLessThan(0.5);
  });
});

describe("resolveShape error and absence handling", () => {
  it("returns undefined for a query that matches nothing", async () => {
    const rec = await resolveShape(GUIDED_SHAPES[0]!, async () => []);
    expect(rec).toBeUndefined();
  });

  it("propagates a finder failure instead of reporting no matches", async () => {
    // The #63 invariant: an error must never be indistinguishable from an absence
    // of data. "The catalog is broken" and "nothing matches your query" are
    // different facts and the second one is a claim we can't support here.
    await expect(
      resolveShape(GUIDED_SHAPES[0]!, async () => {
        throw new Error("catalog unreadable");
      }),
    ).rejects.toThrow("catalog unreadable");
  });

  it("falls back to the top match with an unknown price when nothing is priced", async () => {
    const shape = GUIDED_SHAPES[0]!;
    const rec = (await resolveShape(shape, async () => [
      stub({ instanceType: "x1.unknown", onDemandPrice: undefined }),
      stub({ instanceType: "x2.unknown", onDemandPrice: 0 }),
    ]))!;
    expect(rec.pick.instance.instanceType).toBe("x1.unknown");
    // Reported as unknown, not invented and not zero: zero would render as free.
    expect(rec.pricePerHour).toBeUndefined();
    expect(rec.estimatedTotal).toBeUndefined();
  });

  it("treats a zero price as missing data", async () => {
    const rec = (await resolveShape(GUIDED_SHAPES[0]!, async () => [
      stub({ instanceType: "free.lie", onDemandPrice: 0 }),
      stub({ instanceType: "real.one", onDemandPrice: 1.5 }),
    ]))!;
    // No EC2 type costs nothing per hour, so a zero can only be an artifact —
    // and it would otherwise sort first and win.
    expect(rec.pick.instance.instanceType).toBe("real.one");
    expect(rec.pricePerHour).toBe(1.5);
  });

  it("still recommends a GPU box when a non-GPU shape has no CPU-only match", async () => {
    // Narrowing to CPU-only must not turn into "no recommendation": an unexpected
    // accelerator whose price we can show beats a dead card.
    const cpuShape = GUIDED_SHAPES.find((s) => !s.wantsGpu)!;
    const rec = (await resolveShape(cpuShape, async () => [
      stub({ instanceType: "g9.big", onDemandPrice: 3, gpus: 1 }),
    ]))!;
    expect(rec.pick.instance.instanceType).toBe("g9.big");
  });

  it("marks an estimated price as estimated", async () => {
    const rec = (await resolveShape(GUIDED_SHAPES[0]!, async () => [
      stub({ instanceType: "e1.guess", onDemandPrice: 2, estimatedPrice: true }),
    ]))!;
    // An estimate presented as a price is the same defect as a fabricated one,
    // just smaller — so the flag has to survive to the UI.
    expect(rec.priceIsEstimate).toBe(true);
  });
});

const QUARANTINED = new Set(["p6e-gb200.36xlarge", "p5.4xlarge"]);
const isQuarantined = (r: FindResult) => QUARANTINED.has(r.instance.instanceType);
const usable = (r: FindResult) =>
  typeof r.instance.onDemandPrice === "number" && r.instance.onDemandPrice > 0;

/** A minimal FindResult for the stubbed-finder cases. */
function stub(over: Partial<FindResult["instance"]>): FindResult {
  return {
    instance: {
      instanceType: "t0.test",
      instanceFamily: "t0",
      vcpus: 2,
      memoryMib: 4096,
      architecture: "x86_64",
      ...over,
    },
    reasons: [],
  } as FindResult;
}
