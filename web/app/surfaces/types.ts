// The pluggable-surface contract. Each CLI tool (spawn, truffle, lagotto…) shows
// up in the portal as one ToolSurface. Adding a tool later = write one
// surfaces/<tool>.ts and append it to the registry — the shell never changes.
import type { SessionController } from "../session.js";

export interface SurfaceContext {
  /** The signed-in session (creds in memory, accountId, expiry). */
  session: SessionController;
  /** Build-time + URL-override config (region, role ARN, client id…). */
  config: PortalConfig;
  /** Navigate to another surface by id (updates the hash router). */
  navigate(surfaceId: string): void;
}

/** Returned by mount(); the shell calls dispose() when leaving the surface. */
export interface Disposable {
  dispose(): void;
}

export interface ToolSurface {
  /** Stable id, used in the hash route (#/<id>). */
  id: string;
  /** Human label for the sidebar. */
  label: string;
  /** CSS custom property holding the tool's accent colour, e.g. "--spawn". */
  accent: string;
  /** If true, the shell gates this surface behind sign-in. */
  requiresAuth: boolean;
  /** Render into `host`; return a handle the shell disposes on navigation. */
  mount(host: HTMLElement, ctx: SurfaceContext): Promise<Disposable>;
}

/** Portal configuration, resolved from Vite build-time defaults + URL overrides. */
export interface PortalConfig {
  region: string;
  /**
   * Base URL of the shared dashboard-api (teams, strata catalog, cost-history,
   * Slack). Authenticated via the session's federated creds in an
   * X-AWS-Credentials header. Defaults to https://api.spore.host.
   */
  apiBase: string;
  /** Globus OIDC application (public client) id. */
  globusClientId: string;
  /** The AssumeRoleWithWebIdentity role the browser federates into. */
  roleArn: string;
  /** OAuth redirect URI (this portal's app/ URL). */
  redirectUri: string;
  /**
   * BYOA onboarding (the "connect your AWS account" quick-create). Optional —
   * the onboarding surface degrades to instructions if any are unset.
   */
  onboarding: {
    /** Public HTTPS URL of the CloudFormation onboarding template. */
    templateUrl: string;
    /** Phone-home Lambda execution role ARN the onboarding role trusts. */
    phoneHomeRoleArn: string;
    /** Phone-home Function URL (baked into the template default too). */
    phoneHomeUrl: string;
  };
}
