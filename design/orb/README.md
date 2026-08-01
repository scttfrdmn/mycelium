# Animated hero orb — source

The `.riv` on the homepage is a build artifact. This directory is its source, so
the animation can be edited later instead of being reverse-engineered from a
binary.

| Path | What it is |
|---|---|
| `scene.json` | The whole rig — artboard, layers, mesh, every keyframe |
| `parts/` | The sliced mascot art the scene references |
| `build.mjs` | `scene.json` + `parts/` → `web/assets/brand/spore-orb.riv` |

## Rebuild

```bash
npm i -g rive-mcp-server          # provides the .riv writer
node design/orb/build.mjs         # writes web/assets/brand/spore-orb.riv
```

The build is deterministic: same inputs → byte-identical `.riv`.

## The rig

Artboard `Orb`, 520×560 — taller than wide because the fringe trails well below
the bell. Animation `idle`, 480 frames @ 60fps = **8s, seamless loop**, driven by
state machine `SM`.

- **`bell`, `eyeL`, `eyeR`** — static images on the `root` group.
- **`tentacles`** — a 400×240 image with a **24×5 deformation mesh**. Every
  vertex is keyframed on x; this is what makes the strands sway.
- **`halo`** — a screen-blended radial gradient behind the bell, breathing on
  scaleX/scaleY.
- **`pulse1/2/3`** — small screen-blended droplets that rise and fade, 6 flights
  per loop.
- **`glow-pulse` preset** on the two eyes, staggered.

### Things that will bite you

**No rotation anywhere, deliberately.** Any rotation on `root` or `tentG` reads
as the *whole mascot rotating* rather than the strands swaying — it was the single
most-rejected artifact during art direction. Note that the `float-idle` preset
**bundles a hidden rotation track** (±1.5°·intensity), which is why the float here
is hand-authored as a plain y-bob plus a 0.6% scaleY breath instead. A preset also
can't coexist with a manual track on the same target+property.

**Mesh tearing comes from sign flips, not amplitude.** Adjacent columns
displaced in *opposite* directions shear the quad between them and sever the thin
diagonal strands into dashes. Smooth traveling waves at 26% and even 47% cell
shear render clean; a 42% shear with opposing neighbours does not. So keep the
displacement field smooth across columns — the current mix is three harmonics
(k=2,4,6 over the loop) with per-column amplitude weighting, giving adjacent-strand
correlation ≈0.85 and anti-correlated fringe ends.

**Every ambient track must start and end on the same value** or the loop visibly
jumps. The float is one cycle per loop — the slowest possible at this length. To
slow it further you must lengthen the loop and multiply every other element's
cycle count to keep those rates unchanged (that is exactly why the loop is 8s and
the harmonics are k=2,4,6 rather than 1,2,3).

**`parts/tentacles.png` is cut from the original mascot art, not from
`base.png`.** `riv_slice_image` *erases* each slice from the base layer, so
cropping the fringe out of `base.png` yields a hole-punched image — that produced
a visible hard arc under the bell (8032 missing pixels) until it was re-cut.

**`parts/q_tentacles_256.png` is the quantized fringe** actually used by the
scene (256 colors, 167K → 51K). The fringe is line art and quantizes invisibly;
the **bell is not quantized** because banding shows in its gloss gradient. Full
quantization measured 4× the pixel error for 119KB more savings — not worth it.
`parts/tentacles.png` is kept as the lossless master.

## Renderer constraint

The site loads the **webgl2** Rive runtime, not `canvas`/`canvas-lite`. Both
Canvas2D runtimes clip this animated mesh to its first row and drop the entire
fringe. See `web/assets/vendor/rive/LICENSE-MIT.txt`.

## Verifying a change

`.riv` diffs are meaningless, so check the render:

```bash
cd web && npm run build && python3 -m http.server 8791 --directory dist
```

Then load `http://127.0.0.1:8791/` in a real browser and confirm:

1. The **full fringe** renders — strands reaching well below the bell, not clipped
   at the bell's edge.
2. Individual strands sway; the mascot as a whole does **not** rotate.
3. The loop doesn't jump at the seam (watch one full 8s cycle).
4. No dashes or staircase breaks in the strands (check at 2× zoom).
5. Background is transparent — no dark box on the light theme.
6. With `chrome --headless --disable-gpu` (no WebGL2) the static mark shows
   instead, not an empty gap.
