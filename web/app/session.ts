// The portal's session: the signed-in user's AWS credentials (in memory only —
// never localStorage), the account they resolve to, and an expiry timer that
// warns/tears-down before the STS creds lapse. Surfaces subscribe via onExpiry.
import { GetCallerIdentityCommand, STSClient } from "@aws-sdk/client-sts";
import type { AwsCreds } from "@spore-host/spawn-ts/auth";

/** Fires when creds are within the warn window or already expired. */
export type ExpiryListener = (state: "warning" | "expired") => void;

// Re-auth this long before the hard expiration so an in-flight action doesn't
// die mid-request.
const WARN_BEFORE_MS = 5 * 60 * 1000;

export class SessionController {
  private creds: AwsCreds | null = null;
  private _accountId: string | null = null;
  private _region: string;
  private timer: ReturnType<typeof setTimeout> | null = null;
  private expiryListeners = new Set<ExpiryListener>();

  constructor(region: string) {
    this._region = region;
  }

  get region(): string {
    return this._region;
  }

  get accountId(): string | null {
    return this._accountId;
  }

  get signedIn(): boolean {
    return this.creds !== null;
  }

  /** The live credentials, or null if not signed in. */
  getCreds(): AwsCreds | null {
    return this.creds;
  }

  /**
   * Adopt freshly federated credentials, resolve the account via STS, and arm
   * the expiry timer. Throws if GetCallerIdentity fails (bad creds/region).
   */
  async adopt(creds: AwsCreds): Promise<void> {
    const sts = new STSClient({
      region: this._region,
      credentials: {
        accessKeyId: creds.accessKeyId,
        secretAccessKey: creds.secretAccessKey,
        sessionToken: creds.sessionToken,
      },
    });
    const who = await sts.send(new GetCallerIdentityCommand({}));
    this.creds = creds;
    this._accountId = who.Account ?? null;
    this.armExpiry();
  }

  /** Drop creds + timers (sign-out). */
  clear(): void {
    this.creds = null;
    this._accountId = null;
    if (this.timer) {
      clearTimeout(this.timer);
      this.timer = null;
    }
  }

  /** Subscribe to expiry transitions; returns an unsubscribe fn. */
  onExpiry(fn: ExpiryListener): () => void {
    this.expiryListeners.add(fn);
    return () => this.expiryListeners.delete(fn);
  }

  private armExpiry(): void {
    if (this.timer) clearTimeout(this.timer);
    const exp = this.creds?.expiration?.getTime();
    if (!exp) return; // no expiration → nothing to arm
    const now = performance.timeOrigin + performance.now();
    const warnAt = exp - WARN_BEFORE_MS - now;
    if (warnAt <= 0) {
      this.emit("warning");
      return;
    }
    this.timer = setTimeout(() => {
      this.emit("warning");
      const hardAt = exp - (performance.timeOrigin + performance.now());
      this.timer = setTimeout(() => this.emit("expired"), Math.max(0, hardAt));
    }, warnAt);
  }

  private emit(state: "warning" | "expired"): void {
    for (const fn of this.expiryListeners) fn(state);
  }
}
