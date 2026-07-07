# docs-site — OpenTofu module

Codifies the delivery layer for **docs.spore.host** (the VitePress documentation
site). Mirrors the `dns-updater` / `spore-bot` import-onto-live convention: the
live resources were created imperatively (console/CLI), and this module was
`tofu import`ed onto them, planning to a near-zero diff before it owned anything.

## Why this exists (#404)

`docs.spore.host` went fully dark — **503 on every request** — because the
CloudFront function `spore-host-docs-url-rewrite` (viewer-request) had **corrupt
code**: 141 bytes of binary garbage instead of valid JavaScript, so CloudFront
could not execute it. Republishing a correct function restored the site.

The root cause of the *fragility* was that none of the docs delivery layer was
under source control — the distribution, S3 origin, OAC, and especially the
**function code** were pure console artifacts. A bad manual edit could (and did)
take the whole site down with no reviewable history.

**The primary point of this module is that the CloudFront function code now lives
in `url-rewrite.js` and is the source of truth.** A change to it is a reviewable
diff, and `tofu apply` publishes it — corruption cannot silently ship.

## What it manages

| Resource | Notes |
|----------|-------|
| `aws_cloudfront_function.url_rewrite` | Code from `url-rewrite.js`. Clean URL `/x` → `/x.html`, `/` or dir → `index.html`, extensions pass through. |
| `aws_cloudfront_distribution.docs` | `E1F70PIGPUNRR0`, alias `docs.spore.host`. |
| `aws_cloudfront_origin_access_control.docs` | `E3KJVACTARMGFR` (S3 SigV4, signing always). |
| `aws_s3_bucket.docs` + policy + public-access-block | `spore-host-docs`. OAC-scoped read only. |
| `aws_route53_record.docs` | `docs.spore.host` A-alias → the distribution. |

## Deliberately NOT managed here

- **S3 object content** — the built VitePress site is deployed by
  `.github/workflows/docs.yaml` (`aws s3 sync docs/.vitepress/dist s3://spore-host-docs`).
  Tofu owns the bucket's *shape*, never its objects.

## Changing the URL-rewrite function

1. Edit `url-rewrite.js`.
2. `AWS_PROFILE=spore-host-infra tofu plan` — review the `code` diff.
3. `tofu apply` — publishes to the LIVE stage automatically (`publish = true`).
4. Invalidate if needed:
   `aws cloudfront create-invalidation --distribution-id E1F70PIGPUNRR0 --paths "/*"`.

## Usage

```bash
AWS_PROFILE=spore-host-infra tofu init
AWS_PROFILE=spore-host-infra tofu plan     # should be "No changes"
AWS_PROFILE=spore-host-infra tofu apply
```

State is local and git-ignored (see `.gitignore`), same as the sibling modules.

## Import reference (already done)

```bash
tofu import aws_s3_bucket.docs spore-host-docs
tofu import aws_s3_bucket_public_access_block.docs spore-host-docs
tofu import aws_s3_bucket_policy.docs spore-host-docs
tofu import aws_cloudfront_origin_access_control.docs E3KJVACTARMGFR
tofu import aws_cloudfront_distribution.docs E1F70PIGPUNRR0
tofu import aws_cloudfront_function.url_rewrite spore-host-docs-url-rewrite   # by NAME, not ARN
tofu import aws_route53_record.docs Z0341053304H0DQXF6U4X_docs.spore.host_A
```
