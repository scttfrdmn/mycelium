---
description: "The spore.host portal has one control that decides how much of the interface you see — Guided, Standard, or Expert. What each level shows, and why Guided works with no AI and no credentials."
---

# Portal detail levels

The [portal](https://spore.host/app) has one control in its header, labelled
**Detail**, with three settings. It decides how much of the interface you see.

| Level | What you get |
|---|---|
| **Guided** | One question — "What are you doing?" — a short list to pick from, and the machine, cost and shutdown time chosen for you. |
| **Standard** | The instance search box, the full launch form (type, spot, time limit), and the instance list. |
| **Expert** | Everything above, plus per-instance-type detail: physical cores vs threads, GPU vendor and total VRAM, family, nested-virt support, and whether each price came from a live AWS pull or a hand estimate. |

The setting is **remembered** and applies **across the whole portal**, not
per-page. Changing it re-renders whatever you're looking at.

## Guided mode

Guided mode asks *"What are you doing?"* in ordinary words and answers with one
machine:

> **A small analysis** — Notebooks, scripts, modest data. The usual starting point.
> `t4g.xlarge` — 4 vCPU · 16 GiB
> about $0.54 for 4 hours ($0.1344/hr)

The cost leads with the **total for the whole run**, not the hourly rate. "$0.1344
per hour" is a number a first-time user cannot act on; "about $0.54" is.

Picking one gets a single confirmation screen — machine, region, *"Shuts down
automatically after 4 hours, whatever happens"*, the cost — and one **Start it**
button. `← Something else` goes back without launching.

Every guided launch gets a **time limit** — 4 hours for CPU shapes, 2 for GPU ones
— and none of them use spot. Both are deliberate:

- **The time limit is not optional.** An instance nobody remembered to stop is the
  most expensive mistake this interface makes available. The limit is enforced by
  `spored` *on the machine*, so it holds even if you close the tab — which is
  exactly what the confirmation screen says, because a user who closes the tab
  must not be left guessing whether their instance now runs forever or was killed.
- **Spot is off.** A spot instance can be reclaimed mid-run. That's a good trade
  once you understand it and a baffling one before you do. See
  [Spot instances](/guides/spot-instances) for when to take it.

Guided mode hides the launch form but **never** the instance list. Being able to
start an instance without being able to see or stop it is the one simplification
that would cost real money.

**I know what I need →** at the bottom of the list moves you to Standard. Guided is
a starting point, not a cage.

### Guided mode needs nothing but the portal

No AI, no model access, no credentials, no network. The curated list resolves
against truffle-ts's bundled instance catalog, which ships with the page.

This is the right way round, and it's worth being explicit about why: the
beginner's entry point must not be the thing that breaks first. If the simplest
mode depended on an AI backend, the one path a first-time user takes would be the
one most likely to be broken — and the one nobody developing the portal is ever in,
so nobody would notice.

When an AI advisor *is* available it replaces the fixed list with a free-text
question **in the same slot**. Better answer, same place, and the list is still
there when it isn't. The advisor is never a prerequisite.

Two things the guided picker does that are easy to get wrong:

- **It re-sorts by price.** truffle treats `4 vcpus 16gb` as a *minimum* and ranks
  by its own size preference, so its first result for that query is an
  `r8g.12xlarge` at $2.83/hr. Taking the top hit would quote 21× the right price.
- **It won't hand a non-GPU shape a GPU.** `8 vcpus 128gb` otherwise matches
  `g3.4xlarge`, charging someone who asked for memory for a decade-old accelerator
  they can't use.

## Standard mode

The search box takes plain language — `nvidia h100`, `8 gpus a100`,
`32 vcpus arm` — and the launch form exposes instance type, spot, and the time
limit. This is the level most people stay at. The same catalog and the same
queries are documented at [Finding the right instance](/guides/finding-instances).

## Expert mode

Expert adds, per result, the fields a capacity or topology decision actually turns
on:

```
family            g6
physical cores    24
threads/core      2
memory            196608 MiB
GPU vendor        nvidia
GPU memory        91552 MiB (total)
nested virt       no
price source      live AWS pull
```

`physical cores` matters because an MPI rank count is cores, not vCPUs.
`price source` matters because a few catalog entries are hand estimates rather than
live figures, and an estimate shown as a price is a small lie — expert is the level
that can act on knowing which it is. A field the catalog genuinely doesn't carry is
**omitted**, not rendered as `0` or `—`; a zero where the real answer is "unknown"
states something false about the hardware to the user most likely to act on it.

Standard hides all of this because it's noise when you're comparing two boxes.

## Notes

- The default for a first-time visitor is **Guided**. If the default were Standard,
  the mode built for the least-experienced user would be the one they never see.
- Signing out doesn't reset it. Someone who has set Expert shouldn't be dropped
  back to Guided by an expired session.
- The level is stored in `localStorage` under `spore.disclosure`. It's a
  preference, not a credential — your AWS credentials never leave the tab's memory
  (see [Security, credentials & data flow](/architecture)).
