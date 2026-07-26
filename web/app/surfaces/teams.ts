// The teams surface: create teams and manage membership via the shared
// dashboard-api (/teams…) over the session's federated creds. This is the first
// portal surface with WRITE/mutating ops (POST create, POST/DELETE members,
// DELETE team) — every destructive action is confirm-gated, and the API itself
// enforces owner-only on writes (403 otherwise), so the UI just surfaces that.
//
// Identity note: with account-scoped auth (spawn#445), the caller's team key is
// their verified AWS account id; teammates are added by their IAM user/role ARN.
import type { AwsCreds } from "@spore-host/spawn-ts/auth";
import type { Disposable, SurfaceContext, ToolSurface } from "./types.js";

interface Team {
  team_id: string;
  team_name: string;
  owner_arn: string;
  description?: string;
  created_at: string;
  member_count: number;
  role?: string; // present on the list endpoint (owner | member)
}

interface Member {
  team_id: string;
  member_arn: string;
  role: string;
  joined_at: string;
  invited_by: string;
}

// A valid IAM user/role ARN — mirrors memberARNRe in the API so we reject before
// the round-trip.
const MEMBER_ARN_RE = /^arn:aws:iam::\d{12}:(user|role)\/.{1,256}$/;

function credentialsHeader(creds: AwsCreds): string {
  return btoa(
    JSON.stringify({
      accessKeyId: creds.accessKeyId,
      secretAccessKey: creds.secretAccessKey,
      sessionToken: creds.sessionToken,
    }),
  );
}

export const teamsSurface: ToolSurface = {
  id: "teams",
  label: "Teams",
  accent: "--lagotto",
  requiresAuth: true,

  async mount(host: HTMLElement, ctx: SurfaceContext): Promise<Disposable> {
    const creds = ctx.session.getCreds();
    if (!creds) throw new Error("teams surface mounted without a session");

    const root = document.createElement("div");
    root.className = "teams-surface";
    host.appendChild(root);

    const controller = new AbortController();
    let disposed = false;

    // Thin fetch wrapper: attaches auth, JSON content-type on writes, and the
    // abort signal; returns parsed body + status.
    async function api(
      path: string,
      init?: { method?: string; body?: unknown },
    ): Promise<{ ok: boolean; status: number; data: any }> {
      const headers: Record<string, string> = { "X-AWS-Credentials": credentialsHeader(creds!) };
      let body: string | undefined;
      if (init?.body !== undefined) {
        headers["content-type"] = "application/json";
        body = JSON.stringify(init.body);
      }
      const resp = await fetch(`${ctx.config.apiBase}${path}`, {
        method: init?.method ?? "GET",
        headers,
        body,
        signal: controller.signal,
      });
      let data: any = null;
      try {
        data = await resp.json();
      } catch {
        /* empty/non-JSON body */
      }
      return { ok: resp.ok, status: resp.status, data };
    }

    function errText(status: number, data: any): string {
      if (data?.error) return String(data.error);
      if (status === 401) return "authentication failed";
      if (status === 403) return "you don't have permission for that";
      return `HTTP ${status}`;
    }

    // ── List view ──────────────────────────────────────────────────────────
    async function showList(): Promise<void> {
      if (disposed) return;
      root.replaceChildren();
      root.append(
        el("div", "teams-head", [
          el("h2", "", ["Teams"]),
          el("p", "teams-hint", [
            "Teams share visibility of spawn-managed instances. You own the teams you create; owners manage membership.",
          ]),
        ]),
      );

      const status = el("div", "teams-status", ["loading…"]);
      root.append(status);

      const res = await api("/teams");
      if (disposed) return;
      if (!res.ok) {
        status.className = "teams-status error";
        status.textContent = `Couldn't load teams (${errText(res.status, res.data)}).`;
        return;
      }
      status.remove();

      const teams: Team[] = res.data?.teams ?? [];

      // Create-team form (collapsible).
      root.append(buildCreateForm(showList, api, errText));

      if (teams.length === 0) {
        root.append(el("div", "teams-status", ["You're not in any teams yet — create one above."]));
        return;
      }

      const list = el("div", "teams-list", []);
      for (const t of teams) {
        const card = el("button", "teams-card", []);
        (card as HTMLButtonElement).type = "button";
        card.append(
          el("div", "teams-card-head", [
            spanText("teams-name", t.team_name),
            spanText("teams-role", t.role ?? "member"),
          ]),
          el("div", "teams-card-meta", [
            `${t.member_count} member${t.member_count === 1 ? "" : "s"}`,
          ]),
        );
        if (t.description) card.append(spanText("teams-desc", t.description));
        card.addEventListener("click", () => void showDetail(t.team_id));
        list.append(card);
      }
      root.append(list);
    }

    // ── Detail view ────────────────────────────────────────────────────────
    async function showDetail(teamID: string): Promise<void> {
      if (disposed) return;
      root.replaceChildren();
      const back = el("button", "teams-back", ["← All teams"]);
      (back as HTMLButtonElement).type = "button";
      back.addEventListener("click", () => void showList());
      root.append(back);

      const status = el("div", "teams-status", ["loading…"]);
      root.append(status);

      const res = await api(`/teams/${encodeURIComponent(teamID)}`);
      if (disposed) return;
      if (!res.ok) {
        status.className = "teams-status error";
        status.textContent = `Couldn't load team (${errText(res.status, res.data)}).`;
        return;
      }
      status.remove();

      const team: Team = res.data.team;
      const members: Member[] = res.data.members ?? [];
      const iAmOwner = team.owner_arn === ctx.session.accountId;

      root.append(
        el("div", "teams-head", [
          el("h2", "", [team.team_name]),
          ...(team.description ? [el("p", "teams-hint", [team.description])] : []),
        ]),
      );

      // Members table.
      const table = el("table", "teams-members", []);
      const thead = el("thead", "", []);
      const htr = el("tr", "", []);
      for (const h of ["Member", "Role", "Joined", ""]) htr.append(el("th", "", [h]));
      thead.append(htr);
      table.append(thead);
      const tbody = el("tbody", "", []);
      for (const m of members) {
        const tr = el("tr", "", []);
        tr.append(
          tdText(m.member_arn),
          tdText(m.role),
          tdText(fmtDate(m.joined_at)),
        );
        const actionTd = el("td", "teams-memb-action", []);
        // Owner can remove non-owner members.
        if (iAmOwner && m.role !== "owner") {
          const rm = el("button", "teams-remove", ["Remove"]);
          (rm as HTMLButtonElement).type = "button";
          rm.addEventListener("click", () => void removeMember(teamID, m.member_arn));
          actionTd.append(rm);
        }
        tr.append(actionTd);
        tbody.append(tr);
      }
      table.append(tbody);
      root.append(table);

      // Owner controls: add member + delete team.
      if (iAmOwner) {
        root.append(buildAddMemberForm(teamID, () => void showDetail(teamID), api, errText));
        const danger = el("div", "teams-danger", []);
        const del = el("button", "teams-delete", ["Delete team"]);
        (del as HTMLButtonElement).type = "button";
        del.addEventListener("click", () => void deleteTeam(teamID, team.team_name));
        danger.append(del);
        root.append(danger);
      }
    }

    async function removeMember(teamID: string, memberARN: string): Promise<void> {
      if (!window.confirm(`Remove ${memberARN} from the team?`)) return;
      const res = await api(`/teams/${encodeURIComponent(teamID)}/members/${encodeURIComponent(memberARN)}`, {
        method: "DELETE",
      });
      if (disposed) return;
      if (!res.ok) {
        window.alert(`Couldn't remove member: ${errText(res.status, res.data)}`);
        return;
      }
      void showDetail(teamID);
    }

    async function deleteTeam(teamID: string, name: string): Promise<void> {
      if (!window.confirm(`Delete team "${name}"? This removes all memberships and can't be undone.`)) return;
      const res = await api(`/teams/${encodeURIComponent(teamID)}`, { method: "DELETE" });
      if (disposed) return;
      if (!res.ok) {
        window.alert(`Couldn't delete team: ${errText(res.status, res.data)}`);
        return;
      }
      void showList();
    }

    void showList();

    return {
      dispose() {
        disposed = true;
        controller.abort();
        root.remove();
      },
    };
  },
};

// ── form builders ──────────────────────────────────────────────────────────
type ApiFn = (p: string, i?: { method?: string; body?: unknown }) => Promise<{ ok: boolean; status: number; data: any }>;
type ErrFn = (status: number, data: any) => string;

function buildCreateForm(onCreated: () => void, api: ApiFn, errText: ErrFn): HTMLElement {
  const details = document.createElement("details");
  details.className = "teams-create";
  const summary = document.createElement("summary");
  summary.textContent = "New team";
  details.append(summary);

  const form = document.createElement("form");
  form.className = "teams-form";
  const name = inputEl("text", "Team name", true);
  name.maxLength = 100;
  const desc = inputEl("text", "Description (optional)", false);
  desc.maxLength = 1000;
  const submit = el("button", "teams-submit", ["Create"]) as HTMLButtonElement;
  submit.type = "submit";
  const msg = el("span", "teams-formmsg", []);
  form.append(name, desc, submit, msg);
  details.append(form);

  form.addEventListener("submit", (e) => {
    e.preventDefault();
    const teamName = name.value.trim();
    if (!teamName) {
      msg.textContent = "team name is required";
      msg.className = "teams-formmsg error";
      return;
    }
    submit.disabled = true;
    msg.textContent = "";
    void api("/teams", { method: "POST", body: { team_name: teamName, description: desc.value.trim() } }).then(
      (res) => {
        submit.disabled = false;
        if (!res.ok) {
          msg.textContent = errText(res.status, res.data);
          msg.className = "teams-formmsg error";
          return;
        }
        onCreated();
      },
    );
  });
  return details;
}

function buildAddMemberForm(teamID: string, onAdded: () => void, api: ApiFn, errText: ErrFn): HTMLElement {
  const form = document.createElement("form");
  form.className = "teams-form teams-addmember";
  const arn = inputEl("text", "arn:aws:iam::123456789012:user/name", true);
  const submit = el("button", "teams-submit", ["Add member"]) as HTMLButtonElement;
  submit.type = "submit";
  const msg = el("span", "teams-formmsg", []);
  form.append(arn, submit, msg);

  form.addEventListener("submit", (e) => {
    e.preventDefault();
    const memberArn = arn.value.trim();
    if (!MEMBER_ARN_RE.test(memberArn)) {
      msg.textContent = "enter a valid IAM user or role ARN";
      msg.className = "teams-formmsg error";
      return;
    }
    submit.disabled = true;
    msg.textContent = "";
    void api(`/teams/${encodeURIComponent(teamID)}/members`, { method: "POST", body: { member_arn: memberArn } }).then(
      (res) => {
        submit.disabled = false;
        if (!res.ok) {
          msg.textContent = errText(res.status, res.data);
          msg.className = "teams-formmsg error";
          return;
        }
        arn.value = "";
        onAdded();
      },
    );
  });
  return form;
}

// ── tiny DOM helpers (all text via textContent — API strings are untrusted) ──
function el(tag: string, className: string, children: (Node | string)[]): HTMLElement {
  const node = document.createElement(tag);
  if (className) node.className = className;
  for (const c of children) node.append(typeof c === "string" ? document.createTextNode(c) : c);
  return node;
}

function spanText(className: string, text: string): HTMLElement {
  return el("span", className, [text]);
}

function tdText(text: string): HTMLElement {
  return el("td", "", [text]);
}

function inputEl(type: string, placeholder: string, required: boolean): HTMLInputElement {
  const i = document.createElement("input");
  i.type = type;
  i.placeholder = placeholder;
  i.required = required;
  i.className = "teams-input";
  i.autocomplete = "off";
  return i;
}

function fmtDate(iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  return d.toLocaleDateString(undefined, { year: "numeric", month: "short", day: "numeric" });
}
