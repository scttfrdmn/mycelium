// The lagotto surface: watch for EC2 instance-type capacity and say when it
// appears. This is the ninth tool in the portal and the first added AFTER the
// shell was built — the whole point of the ToolSurface contract. Adding it
// touched exactly this file, one registry entry, a CSS block, and one IAM action.
//
// The decision logic is NOT reimplemented here: @spore-host/lagotto-ts/live's
// CapacityWatcher drives check/poll and the pure matcher decides what counts as a
// match (price cap, AZ pin, spot vs on-demand), identical to the `lagotto` CLI.
// What the browser must supply is the CapacityFinder seam — lagotto-ts owns the
// interface and deliberately ships no AWS client.
//
// Why a portal-local finder instead of truffle-ts's AwsLiveFinder: capacity
// watching needs AZ-LEVEL offering data ("is p5.48xlarge offered in us-east-1b
// right now?"), and truffle-ts's live finder is region-agnostic — its
// InstanceType carries no region/availableAZs (results are deduped across
// regions) and its getSpotPricing is still a stub (truffle-ts#18). So this file
// implements the seam directly over ec2:DescribeInstanceTypeOfferings (+
// DescribeSpotPriceHistory for a Spot watch), the same two calls Go truffle's
// getAvailabilityZones / GetSpotPricing make. When truffle-ts#18 lands and its
// finder grows per-region AZs, this can collapse to lagotto-ts's
// truffleFinderAdapter.
import {
  DescribeInstanceTypeOfferingsCommand,
  DescribeSpotPriceHistoryCommand,
  EC2Client,
  type _InstanceType,
} from "@aws-sdk/client-ec2";
import { CapacityWatcher, type CapacityFinder, type FinderInstanceType, type FinderSpotPrice } from "@spore-host/lagotto-ts/live";
import type { MatchResult, Watch } from "@spore-host/lagotto-ts";
import { onDemandPrice } from "@spore-host/truffle-ts";
import type { Disposable, SurfaceContext, ToolSurface } from "./types.js";

/** Poll cadence offered in the UI. A browser tab is a short-lived watcher. */
const INTERVALS = [
  { label: "30s", ms: 30_000 },
  { label: "1m", ms: 60_000 },
  { label: "5m", ms: 300_000 },
] as const;

export const lagottoSurface: ToolSurface = {
  id: "lagotto",
  label: "Watch capacity",
  accent: "--lagotto",
  requiresAuth: true,

  async mount(host: HTMLElement, ctx: SurfaceContext): Promise<Disposable> {
    const creds = ctx.session.getCreds();
    if (!creds) throw new Error("lagotto surface mounted without a session");
    const region = ctx.session.region;
    const ec2 = new EC2Client({
      region,
      credentials: {
        accessKeyId: creds.accessKeyId,
        secretAccessKey: creds.secretAccessKey,
        sessionToken: creds.sessionToken,
      },
    });
    const watcher = new CapacityWatcher({ finder: portalCapacityFinder(ec2, region) });

    const root = document.createElement("div");
    root.className = "lagotto-surface";
    root.innerHTML = `
      <div class="lagotto-head">
        <h2>Watch for capacity</h2>
        <p class="lagotto-hint">Scarce instance types (<code>p5.*</code>, <code>trn2.*</code>)
          come and go. Describe what you want and this polls
          <b>${escapeHtml(region)}</b> until it appears — the same matching the
          <code>lagotto</code> CLI does, running in this tab against your own account.</p>
      </div>

      <form class="lagotto-form">
        <div class="lagotto-fields">
        <div class="lagotto-field lagotto-field-wide">
          <label for="lg-pattern">Instance types</label>
          <input id="lg-pattern" class="lagotto-pattern" type="text" value="p5.*"
                 placeholder="p5.*" autocomplete="off" spellcheck="false" required />
          <span class="lagotto-fieldhint">Glob (<code>p5.*</code>) or a regex</span>
        </div>
        <div class="lagotto-field">
          <label for="lg-maxprice">Max $/hr</label>
          <input id="lg-maxprice" class="lagotto-maxprice" type="number" min="0" step="0.01"
                 placeholder="any" />
          <span class="lagotto-fieldhint">blank = no cap</span>
        </div>
        <div class="lagotto-field">
          <label for="lg-azs">Only these AZs</label>
          <input id="lg-azs" class="lagotto-azs" type="text" placeholder="any"
                 autocomplete="off" spellcheck="false" />
          <span class="lagotto-fieldhint">comma-separated, in preference order</span>
        </div>
        <div class="lagotto-field">
          <label for="lg-interval">Check every</label>
          <select id="lg-interval" class="lagotto-interval">
            ${INTERVALS.map((i) => `<option value="${i.ms}"${i.ms === 60_000 ? " selected" : ""}>${i.label}</option>`).join("")}
          </select>
        </div>
        </div>
        <div class="lagotto-controls">
        <label class="lagotto-check">
          <input class="lagotto-spot" type="checkbox" /> Spot capacity (price by AZ)
        </label>
        <div class="lagotto-actions">
          <button type="submit" class="lagotto-start">Start watching</button>
          <button type="button" class="lagotto-stop" hidden>Stop</button>
          <button type="button" class="lagotto-once">Check once</button>
        </div>
        </div>
      </form>

      <div class="lagotto-result" aria-live="polite"></div>
      <ol class="lagotto-log" aria-live="polite"></ol>`;
    host.appendChild(root);

    const form = root.querySelector<HTMLFormElement>(".lagotto-form")!;
    const patternEl = root.querySelector<HTMLInputElement>(".lagotto-pattern")!;
    const maxPriceEl = root.querySelector<HTMLInputElement>(".lagotto-maxprice")!;
    const azsEl = root.querySelector<HTMLInputElement>(".lagotto-azs")!;
    const intervalEl = root.querySelector<HTMLSelectElement>(".lagotto-interval")!;
    const spotEl = root.querySelector<HTMLInputElement>(".lagotto-spot")!;
    const startBtn = root.querySelector<HTMLButtonElement>(".lagotto-start")!;
    const stopBtn = root.querySelector<HTMLButtonElement>(".lagotto-stop")!;
    const onceBtn = root.querySelector<HTMLButtonElement>(".lagotto-once")!;
    const result = root.querySelector<HTMLElement>(".lagotto-result")!;
    const log = root.querySelector<HTMLElement>(".lagotto-log")!;

    let aborter: AbortController | null = null;

    /** Read the form into a lagotto Watch. Throws on an invalid pattern. */
    function readWatch(): Watch {
      const azs = azsEl.value
        .split(",")
        .map((s) => s.trim())
        .filter(Boolean);
      const maxPrice = maxPriceEl.value.trim() === "" ? undefined : Number(maxPriceEl.value);
      return {
        // The browser holds no watch store, so the id is just for display.
        watchId: `tab-${region}-${patternEl.value.trim()}`,
        instanceTypePattern: patternEl.value.trim(),
        regions: [region],
        ...(azs.length ? { availabilityZones: azs } : {}),
        ...(spotEl.checked ? { spot: true } : {}),
        ...(maxPrice !== undefined && Number.isFinite(maxPrice) && maxPrice > 0 ? { maxPrice } : {}),
      };
    }

    function setRunning(running: boolean): void {
      startBtn.hidden = running;
      stopBtn.hidden = !running;
      onceBtn.disabled = running;
      for (const el of [patternEl, maxPriceEl, azsEl, intervalEl, spotEl]) el.disabled = running;
    }

    function appendLog(text: string): void {
      const li = document.createElement("li");
      li.textContent = text; // AWS-derived strings → textContent
      log.prepend(li);
      // Keep the tail bounded; a 30s watch left open all afternoon is ~500 checks.
      while (log.children.length > 50) log.lastElementChild!.remove();
    }

    function showMatch(m: MatchResult, spot: boolean): void {
      const azs = m.candidateAzs.length ? m.candidateAzs.join(", ") : "—";
      result.className = "lagotto-result found";
      result.innerHTML = `
        <div class="lagotto-match">
          <div class="lagotto-match-head">
            <span class="lagotto-match-type">${escapeHtml(m.instanceType)}</span>
            <span class="lagotto-match-kind">${spot ? "Spot" : "on-demand"}</span>
            <span class="lagotto-match-price">$${m.price.toFixed(4)}/hr</span>
          </div>
          <dl class="lagotto-match-meta">
            <dt>Region</dt><dd>${escapeHtml(m.region)}</dd>
            <dt>Zone</dt><dd>${escapeHtml(m.availabilityZone || "—")}</dd>
            <dt>Also offered in</dt><dd>${escapeHtml(azs)}</dd>
          </dl>
          <p class="lagotto-match-next">Capacity is available now — launch it from
            <a href="#/instances">Instances</a>. Availability can vanish between this
            check and a launch; retry the next zone above on
            <code>InsufficientInstanceCapacity</code>.</p>
        </div>`;
    }

    function showError(msg: string): void {
      result.className = "lagotto-result error";
      result.textContent = msg;
    }

    async function checkOnce(): Promise<void> {
      let watch: Watch;
      try {
        watch = readWatch();
      } catch (err) {
        showError((err as Error).message);
        return;
      }
      onceBtn.disabled = true;
      result.className = "lagotto-result";
      result.textContent = "checking…";
      try {
        const m = await watcher.check(watch);
        if (m) showMatch(m, Boolean(watch.spot));
        else {
          result.className = "lagotto-result";
          result.textContent = `No matching capacity in ${region} right now.`;
        }
      } catch (err) {
        showError(describeAwsError(err));
      } finally {
        onceBtn.disabled = false;
      }
    }

    async function startWatching(): Promise<void> {
      let watch: Watch;
      try {
        watch = readWatch();
      } catch (err) {
        showError((err as Error).message);
        return;
      }
      const intervalMs = Number(intervalEl.value);
      aborter = new AbortController();
      setRunning(true);
      log.replaceChildren();
      result.className = "lagotto-result";
      result.textContent = `Watching ${watch.instanceTypePattern} in ${region}…`;
      try {
        const m = await watcher.poll(watch, {
          intervalMs,
          signal: aborter.signal,
          onCheck: (r, n) => {
            appendLog(r ? `check ${n}: found ${r.instanceType} in ${r.availabilityZone || r.region}` : `check ${n}: no capacity yet`);
          },
        });
        if (m) showMatch(m, Boolean(watch.spot));
        else if (!aborter.signal.aborted) {
          result.className = "lagotto-result";
          result.textContent = "Watch ended without a match.";
        }
      } catch (err) {
        showError(describeAwsError(err));
      } finally {
        aborter = null;
        setRunning(false);
      }
    }

    function stopWatching(): void {
      aborter?.abort();
      result.className = "lagotto-result";
      result.textContent = "Stopped.";
    }

    const onSubmit = (e: Event) => {
      e.preventDefault();
      void startWatching();
    };
    const onOnce = () => void checkOnce();
    form.addEventListener("submit", onSubmit);
    stopBtn.addEventListener("click", stopWatching);
    onceBtn.addEventListener("click", onOnce);

    // Federated creds are what the EC2 client signs with — a poll that outlives
    // them would just log AuthFailure every interval, so stop and say why.
    const offExpiry = ctx.session.onExpiry((state) => {
      if (state === "expired") {
        aborter?.abort();
        showError("Session expired — sign in again to keep watching.");
      }
    });

    return {
      dispose() {
        offExpiry();
        aborter?.abort();
        form.removeEventListener("submit", onSubmit);
        stopBtn.removeEventListener("click", stopWatching);
        onceBtn.removeEventListener("click", onOnce);
        ec2.destroy();
        root.remove();
      },
    };
  },
};

// ── The CapacityFinder seam, implemented over the EC2 API ─────────────────────

/** Cap on instance types carried into a Spot-pricing query (API + latency guard). */
const SPOT_QUERY_CAP = 60;

/** Spot lookback: enough history that every AZ has reported at least once. */
const SPOT_LOOKBACK_MS = 60 * 60 * 1000;

/**
 * lagotto's CapacityFinder over one region, built from the two EC2 calls Go
 * truffle uses: DescribeInstanceTypeOfferings at AZ granularity for "what is
 * offered where", and DescribeSpotPriceHistory for a Spot watch's prices.
 *
 * On-demand prices come from truffle-ts's static table (`onDemandPrice`), NOT the
 * Pricing API — the portal's federated role has no `pricing:*`, and the price
 * only gates the watch's maxPrice comparison. It's a us-east-1 "as of" estimate,
 * so a maxPrice on an on-demand watch is approximate; a Spot watch prices live.
 */
function portalCapacityFinder(ec2: EC2Client, region: string): CapacityFinder {
  return {
    async search(matcher: RegExp): Promise<FinderInstanceType[]> {
      // AZ-level offerings, so a match can name the zone to launch in. EC2's
      // instance-type filter takes shell wildcards, so a glob pattern is pushed
      // server-side (a region has ~800 types × ~6 AZs otherwise); the regex is
      // still applied locally, which is authoritative.
      const wildcard = toEc2Wildcard(matcher.source);
      const azsByType = new Map<string, string[]>();
      let token: string | undefined;
      do {
        const page = await ec2.send(
          new DescribeInstanceTypeOfferingsCommand({
            LocationType: "availability-zone",
            ...(wildcard ? { Filters: [{ Name: "instance-type", Values: [wildcard] }] } : {}),
            MaxResults: 1000,
            NextToken: token,
          }),
        );
        for (const o of page.InstanceTypeOfferings ?? []) {
          if (!o.InstanceType || !o.Location) continue;
          if (!matcher.test(o.InstanceType)) continue;
          const list = azsByType.get(o.InstanceType);
          if (list) list.push(o.Location);
          else azsByType.set(o.InstanceType, [o.Location]);
        }
        token = page.NextToken;
      } while (token);

      return Array.from(azsByType, ([instanceType, azs]) => ({
        instanceType,
        region,
        onDemandPrice: onDemandPrice(instanceType),
        // Sorted so the AZ a match reports is stable across checks rather than
        // following the API's page order.
        availableAZs: azs.sort(),
      }));
    },

    async getSpotPricing(instances: FinderInstanceType[]): Promise<FinderSpotPrice[]> {
      const types = Array.from(new Set(instances.map((i) => i.instanceType))).slice(0, SPOT_QUERY_CAP);
      if (types.length === 0) return [];
      const page = await ec2.send(
        new DescribeSpotPriceHistoryCommand({
          // The SDK types InstanceTypes as its generated enum; these names come
          // from a live DescribeInstanceTypeOfferings response, so they're valid
          // EC2 type names even when newer than the SDK's enum.
          InstanceTypes: types as _InstanceType[],
          ProductDescriptions: ["Linux/UNIX"],
          StartTime: new Date(Date.now() - SPOT_LOOKBACK_MS),
        }),
      );
      // History is a time series per (type, AZ); keep only the LATEST point for
      // each, which is the current price. Mirrors Go truffle's default path.
      const latest = new Map<string, { at: number; price: FinderSpotPrice }>();
      for (const p of page.SpotPriceHistory ?? []) {
        if (!p.InstanceType || !p.AvailabilityZone || !p.SpotPrice) continue;
        const spotPrice = Number.parseFloat(p.SpotPrice);
        if (!Number.isFinite(spotPrice)) continue;
        const at = p.Timestamp?.getTime() ?? 0;
        const key = `${p.InstanceType}@${p.AvailabilityZone}`;
        const prev = latest.get(key);
        if (prev && prev.at >= at) continue;
        latest.set(key, {
          at,
          price: { instanceType: p.InstanceType, region, spotPrice, availabilityZone: p.AvailabilityZone },
        });
      }
      return Array.from(latest.values(), (v) => v.price);
    },
  };
}

/**
 * The EC2 `instance-type` filter value for a compiled pattern, or null when the
 * pattern can't be expressed as a shell wildcard (a real regex) and every type
 * must be fetched and filtered locally.
 *
 * lagotto compiles a glob to an anchored regex (`p5.*` → `^p5\..*$`), so this
 * inverts exactly that shape and bails on anything else.
 */
function toEc2Wildcard(source: string): string | null {
  const m = /^\^(.*)\$$/.exec(source);
  if (!m) return null;
  const body = m[1]!;
  // Only the escapes lagotto's glob→regex conversion produces: `\.` and `.*`.
  if (/[[\]()+?{}|^$]/.test(body.replace(/\\\./g, "").replace(/\.\*/g, ""))) return null;
  if (/(?<!\\)\.(?!\*)/.test(body)) return null; // a bare `.` wildcard — not a glob
  return body.replace(/\.\*/g, "*").replace(/\\\./g, ".");
}

/** AWS SDK errors are noisy; surface the actionable part. */
function describeAwsError(err: unknown): string {
  const e = err as { name?: string; message?: string };
  if (e?.name === "UnauthorizedOperation" || e?.name === "AccessDenied" || e?.name === "AccessDeniedException") {
    return "Your role can't read instance-type offerings in this account (needs ec2:DescribeInstanceTypeOfferings).";
  }
  if (e?.name === "AuthFailure" || e?.name === "ExpiredTokenException") {
    return "Credentials are no longer valid — sign in again.";
  }
  return e?.message ?? String(err);
}

function escapeHtml(s: string): string {
  return s.replace(
    /[&<>"']/g,
    (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" })[c]!,
  );
}
