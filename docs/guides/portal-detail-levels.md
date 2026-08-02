---
description: "The spore.host portal has one control that decides how much of the interface you see — Guided, Standard, or Expert. What each level shows on each page, and why Guided works with no AI and no credentials."
---

# Portal detail levels

The [portal](https://spore.host/app) has one control in its header, labelled
**Mode**, with three settings side by side. It decides how much of the interface
you see.

| Level | What you get |
|---|---|
| **Guided** | One question — "What are you doing?" — a short list to pick from, and the machine, cost and shutdown time chosen for you. |
| **Standard** | The instance search box, the full launch form (type, spot, time limit), and the instance list. |
| **Expert** | Everything above, plus per-instance-type detail: physical cores vs threads, GPU vendor and total VRAM, family, nested-virt support, and whether each price came from a live AWS pull or a hand estimate. |

The setting is **remembered** and applies **across the whole portal**, not
per-page. Changing it re-renders whatever you're looking at.

It's three visible buttons rather than a dropdown, because the levels are an
**ordered** scale and a collapsed control showing one of them hides that. The
current level's one-line description sits beside the buttons where you can read it
before choosing — it used to be a tooltip on each dropdown option, which Safari
never rendered at all.

Changing the mode **rebuilds the page you're on**, so anything the page was holding
would be lost. Mostly it isn't — your truffle query, the cost window and table view,
the team you had open, and a capacity watch's settings all live in the address bar
(`#/truffle?q=nvidia+h100`) and come back at the new level, which also makes them
bookmarkable and shareable. Two things genuinely stop, because they're live rather
than replayable: see [what a Mode change keeps](#what-changing-mode-does-and-doesn-t-cost-you).

Raising the level from inside a page — the **Show me all the options →** buttons —
puts one dismissible line at the top of the window saying what changed and that
**Mode** in the header is where to change it back. One click otherwise silently
rewrites a persistent, portal-wide setting, and finding the way back would need
exactly the knowledge those buttons exist to not require.

## Guided mode

Guided mode asks *"What are you doing?"* in ordinary words and answers with one
machine:

> **A small analysis** — Notebooks, scripts, modest data. The usual starting point.
> `t4g.xlarge` — 4 vCPU · 16 GiB
> about $0.54 for 4 hours ($0.1344/hr)

The cost leads with the **total for the whole run**, not the hourly rate. "$0.1344
per hour" is a number a first-time user cannot act on; "about $0.54" is.

Picking one gets a single confirmation screen — machine, region, when it shuts
itself down, the cost — and one **Start it** button. `← Something else` goes back
without launching.

Every guided launch gets **two** limits, and they're both named on that screen
because they fail differently:

- **A time limit** — 4 hours for CPU shapes, 2 for GPU ones. Enforced by `spored`
  *on the machine*, so it holds even if you close the tab. It does **not** hold if
  the daemon never starts, which is why the instance list flags machines that
  outlived their limit so you can stop them. The confirmation screen says this
  rather than promising the shutdown is unconditional — the portal ships a banner
  for exactly the case such a promise would deny.
- **A spend cap**, set above the expected cost of the run. Derived from the
  instance's tags rather than from anything running on it, so it survives the
  daemon failing. It also turns the cost figure in the instance list into a meter
  against a ceiling instead of a bare number.

**Spot is off**, and the confirmation now says so along with what that costs. A
spot instance can be reclaimed mid-run: a good trade once you understand it and a
baffling one before you do, but paying roughly three times spot's price shouldn't be
silent either. See [Spot instances](/guides/spot-instances) for when to take it.

The cost figure is **compute only** — no EBS volume, no data transfer — and says
so. It's priced from the catalog's `us-east-1` figures, so outside that region the
confirmation adds that your region may differ.

If a machine has **no price** in the catalog, **Start it** stays disabled until you
tick a box saying you know you're launching something you can't be quoted for. The
shapes that land there are the accelerator ones: types with no on-demand row are
generally the $30–100/hr machines, so it's the one place where an accidental click
is most expensive.

Guided mode hides the launch form but **never** the instance list. Being able to
start an instance without being able to see or stop it is the one simplification
that would cost real money.

**I know what I need →** at the bottom of the list moves you to Standard. Guided is
a starting point, not a cage. Picking a shape on the **Find instances** page — which
needs no sign-in and so can't launch anything — carries that choice to the
Instances page rather than making you choose again.

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

## What changes on each page

Not every page has three versions, and that's deliberate — a page with no controls,
no writes and no jargon has nothing to hide and nothing to withhold. Where a page
isn't listed, it looks the same at every level.

| Page | Guided | Standard | Expert |
|---|---|---|---|
| **Instances** | The curated picker instead of the launch form. The instance list is always shown. | Full launch form | + |
| **Find instances** | The curated picker instead of the query box | Query box | + per-result hardware detail and price provenance |
| **Cost history** | Same chart and numbers, plainer labels, and a link to Instances to go stop something | Same chart, shorter labels | + compute/storage/network breakdown, window total, peak hour, a 1-year window, and where the series came from |
| **Teams** | **Read-only.** Your teams and their members, with no forms. | Create teams, add and remove members, delete a team | + team ids, full ARNs, created dates, and who invited whom |
| **Watch capacity** | A short list of things worth waiting for, instead of the pattern, price cap, zone and cadence fields. Picking one watches that whole instance family. | The fields, filled in by hand | + |
| **Terminal** | A list of your running machines to pick from | + **Connect to an id instead →**, for something spawn didn't launch | + |
| **Connect account** | Technical detail collapsed | Collapsed | Expanded — what the IAM role can do, before the button rather than after |

Two rules held throughout:

- **Nothing disappears from the sidebar.** Hiding a page at Guided would mean a user
  told "check Teams" can neither find it nor tell it exists. Guided's promise is
  "you can't hurt yourself here", not "fewer features" — so Teams goes read-only
  rather than absent, with a **Manage teams →** button to move up a level.
- **Cost history hides nothing at any level**, including its **Table** button. That
  button is the chart's non-visual equivalent, not a density control, and the people
  who need it are the least likely to have raised the mode.

### Three pages need no account at all

**Find instances**, **Software catalog** and **Connect account** all work signed out.
That isn't a detail-level decision — it's the same principle one axis over: the pages
that answer *"what is this and what would it cost me?"* are the ones a first-time
visitor reaches first, and putting them behind the account they don't have yet inverts
the order.

The **Software catalog** was gated until recently, and for no better reason than that
the endpoint serving it sat behind the API's authentication check. The list is the same
five environment formations for everybody — the API's own handler takes no arguments
and reads nothing per-account — so signing in bought nothing and cost the one visitor
most likely to be browsing. It's now readable by anyone.

### Waiting for capacity, without knowing instance-type names

**Watch capacity** is the page where the split matters most, not least. Its fields
ask for a glob or regular expression over instance-type names, a comma-separated
list of availability zones, and a price cap in dollars per hour — every one of them
only writable by someone who already knows the answer. And the person who *needs*
this page is by definition someone whose launch just failed for capacity, so `p5.*`
is exactly the string they don't have at that moment.

So Guided asks *"What are you waiting for?"* and offers a short list instead:

> **A GPU for a large training run** — H100-class. The hardest capacity to get, and
> the usual reason to watch.
> `p5.48xlarge` — 192 vCPU · 2048 GiB · 8× H100
> $55.04/hr once you launch it — about $1,321 a day. Watching costs nothing.

Four things about that list are deliberate:

- **It is not the launch list.** You don't wait for a `t4g.xlarge`. Offering "A small
  analysis" here would offer a poll that succeeds on its first check every time,
  which teaches you the page does nothing. Everything on this list is hardware that
  is genuinely and routinely unavailable — the only reason the page exists.
- **The big GPU leads.** The launch list is ordered cheapest-first because the cheap
  answer is usually right for someone choosing what to run. This one is read by
  someone who already knows they want the scarce thing, so it leads with that, and
  the easier-to-find alternatives follow — somewhere to go when you're looking at
  $55/hr and an empty log.
- **The cost line is a rate, not a total.** "About $110 for 2 hours" describes a run,
  and this page starts no run — it starts a poll, which is free. What the figure is
  *for* is deciding whether you want the thing at all, so it's the hourly rate plus
  what a day of it costs, and it says the watching itself is free. A family with no
  listed price says so: those are the most expensive machines AWS rents, and showing
  nothing there would read as free.
- **It watches the whole family, not the one machine.** Picking the H100 card watches
  `p5.*`, not `p5.48xlarge`. You're waiting for capacity, and a `p5.4xlarge` coming
  free while the 48xlarge is still full is a match you want to hear about.

When a match comes in, Guided tells you the type and the zone and then points at
**Mode** rather than at the Instances page — because Guided's Instances page offers
five launch shapes and only the H100 one is on this list. Telling someone who just
waited for a B200 to "launch it from Instances" would send them to a picker that
can't. **Let me name the instance types →** moves up to Standard at any point.

### Opening a shell without knowing an instance id

**Terminal** used to ask for one thing: an instance id, typed into an empty box.
That is a fair description of what SSM needs and a poor description of what anyone
has. Getting an `i-0123456789abcdef0` meant going to Instances, copying it, and
coming back — so a page that never read **Mode** at all was Expert-only without ever
saying so.

It now lists your machines and you pick one:

> `trial-run — t4g.xlarge`
> `overnight-fit — g5.2xlarge (spot)`

Only **running** machines are offered. A stopped one fails inside SSM with a message
about the agent, which reads as a broken portal rather than a parked machine.

The list is the machines **spawn launched** — it comes from the same
`spawn:managed=true` filter the Instances page uses, applied by AWS rather than by
the page. So the default set of things you can open a shell into is the set spawn is
responsible for. **Connect to an id instead →** takes anything you can name, because
a shell into a machine the portal didn't launch is a legitimate thing to want; it's
hidden at Guided, where an instance id isn't something you have and offering to take
one is a dead end dressed as an option.

One exception, and it's the case that matters: if the list **can't be loaded**, the
id field appears at Guided too. A control you may not understand beats a page with no
way forward. The page also distinguishes *"you have no machines"* from *"we couldn't
ask"* — reporting the first when it only knows the second sends you hunting for
machines you have.

**This is a usability fix that also narrowed a real permission.** The old box
validated ids with a pattern — it checked that a string *looked* like an instance id,
which is not the same as checking you're allowed to reach it. The IAM role scoped
stopping and terminating to `spawn:managed=true` but left `ssm:StartSession`
unscoped, so a portal session could open a root-capable shell on any SSM-registered
machine in the account while being correctly denied permission to *stop* that same
machine. A shell is at least as powerful as a stop. The role now scopes
`ssm:StartSession` the same way, in AWS, where it's enforced — the picker is the
part you can see, not the part doing the enforcing. If you onboarded an account with
the CloudFormation template, redeploy it to pick this up.

## What changing Mode does and doesn't cost you

Changing Mode rebuilds the page you're on — that is how the new mode takes effect.
Anything the page was holding in memory would go with it, so the things worth keeping
are written into the address bar and restored on the way back:

| Page | Kept across a Mode change |
|---|---|
| **Find instances** | Your query, re-run at the new level |
| **Cost history** | The time window, and whether you were reading the table |
| **Teams** | The team you had open |
| **Watch capacity** | What was being watched — the instance types, price cap, zones and cadence, whether you typed them or picked a card |

Two things genuinely stop, because they are live rather than replayable:

- **A capacity watch.** Its settings come back and you get a **Resume watching**
  button, but it does not restart by itself. A watch polls your own AWS account on
  a timer, and a page you didn't ask to load — a bookmark opened the next morning —
  should not start spending on your behalf.
- **A terminal session.** The connection is gone; reconnect from the page.

Both say so before you touch the control, and the watch says so again afterwards.

## Notes

- The default for a first-time visitor is **Guided**. If the default were Standard,
  the mode built for the least-experienced user would be the one they never see.
- Signing out doesn't reset it. Someone who has set Expert shouldn't be dropped
  back to Guided by an expired session.
- The level is stored in `localStorage` under `spore.disclosure`. It's a
  preference, not a credential — your AWS credentials never leave the tab's memory
  (see [Security, credentials & data flow](/architecture)).
- Two pages warn you that changing **Mode** ends something live — a capacity watch,
  or a connected terminal. Both take the name from one constant, because when the
  control was renamed from "Detail" those two warnings were left pointing at a
  control that no longer existed. Being told a running session dies when you touch
  something you can't find is worse than not being warned.
