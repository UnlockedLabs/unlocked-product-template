# unlocked-product-template

The starting point for every Unlocked product repo (EN-202). Create a new
product from it with GitHub's **Use this template** button, then follow
[New product checklist](#new-product-checklist).

It gives a product:

- **One Go binary, one image:** `app serve` runs the API, `app worker` runs the
  background worker.
- **`GET <PATH_PREFIX>/api/version`**, the same JSON as unlocked-core's and the
  tutor's: `{"version": "...", "commit": "...", "branch": "..."}`.
- **`GET <PATH_PREFIX>/api/healthcheck`**, which returns `OK`, the same as
  core's. Point the Deployment's readiness probe at it.
- **Routing under a path prefix:** every route lives under `PATH_PREFIX`, the
  path ingress sends this product's requests to.
- **CI:** tests and golangci-lint on every push, conventional-commit checks,
  and an image pushed to ECR from `main` and from each release.
- **release-please:** versions and the changelog come from the commit messages.

There's no Kubernetes chart here. Every product's templates live in the
`unlocked-platform` chart (EN-183).

## Run it locally

Needs Go 1.26 and Docker.

```sh
make run      # API on :8080, under /<repo name>
make worker   # the worker
make test     # go test -race
make lint     # golangci-lint, in Docker
make build    # the image, named after the repo
```

```sh
curl localhost:8080/<repo name>/api/version
```

Settings, all from the environment:

| Variable | Default | What it does |
|---|---|---|
| `ADDR` | `:8080` | Where the API listens |
| `PATH_PREFIX` | none (root) | The path every route lives under, such as `/example` |
| `APP_VERSION`, `GIT_COMMIT`, `GIT_BRANCH` | `unknown` | What `/api/version` reports. CI bakes them into the image. |

## How to release

1. Merge PRs to `main` with conventional-commit titles (`feat: ...`,
   `fix: ...`). Each merge pushes `:sha-<short>` and `:latest` to ECR.
2. release-please keeps a `chore(main): release X.Y.Z` PR open with the
   changelog. A `feat` bumps the minor version, a `fix` the patch (before
   1.0.0, a breaking change bumps the minor version).
3. Merge that PR when you want a release. It tags `vX.Y.Z`, and CI pushes
   `:X.Y.Z` to ECR.

## How to deploy

In `unlocked-platform`:

1. **First time only:** add the product's templates to the chart behind
   `components.<product>`: a Deployment for `serve` with a readiness probe on
   `<PATH_PREFIX>/api/healthcheck` (and one for `worker` if it has background
   jobs), a Service, ingress paths for its `PATH_PREFIX`, and a
   per-revision migrate Job if it has tables.
2. Pin the release in each site file (`images.<product>.tag: X.Y.Z`) and
   deploy through the deploy workflow. Only staging may use `latest`.

## New product checklist

Replace `<repo>` with the new repo's name, such as `unlocked-open-content`.

1. **Create the repo** from this template under UnlockedLabs.
2. **Rename the module:**
   ```sh
   grep -rl unlocked-product-template --include='*.go' --include=go.mod --include=README.md . \
     | xargs sed -i 's#unlocked-product-template#<repo>#g'
   ```
3. **Branch protection on `main`:** copy the template's `protect-main`
   ruleset (PRs with one approval, and the `vet and test`, `golangci-lint`,
   `conventional` and `pr title` checks), in bash:
   ```bash
   id=$(gh api repos/UnlockedLabs/unlocked-product-template/rulesets --jq '.[] | select(.name=="protect-main") | .id')
   gh api repos/UnlockedLabs/unlocked-product-template/rulesets/$id \
     --jq '{name, target, enforcement, conditions, rules, bypass_actors}' \
     | gh api -X POST repos/UnlockedLabs/<repo>/rulesets --input -
   ```
   `.github/CODEOWNERS` asks `@UnlockedLabs/reviewers` to review every PR, so
   check the team can write to the repo (`maintain` or `push`):
   ```sh
   gh api repos/UnlockedLabs/<repo>/teams --jq '.[] | "\(.slug) \(.permission)"'
   ```
4. **Release bot:** install the unlocked-release-bot GitHub App on the repo.
   Add the variable `RELEASE_APP_CLIENT_ID` and the secret
   `RELEASE_APP_PRIVATE_KEY`.
5. **ECR:** create an ECR repository named `<repo>` in `us-west-2`.
6. **AWS access for CI:** add the repo to the trust policy of the
   `GithubActionsBuildPushEcr` role. New repos get the immutable OIDC subject
   `repo:UnlockedLabs@<owner id>/<repo>@<repo id>:*` (EN-187); check the
   prefix with `gh api repos/UnlockedLabs/<repo>/actions/oidc/customization/sub`
   and the ids with `gh api repos/UnlockedLabs/<repo> --jq '.owner.id, .id'`.
   Then add the role's ARN as the secret `AWS_IAM_ROLE`.
7. **Private Go module?** If other repos import it, set
   `GOPRIVATE=github.com/UnlockedLabs/*` where it's built, and give CI a read
   token.
8. **README:** replace this one with what the product is, how to run it
   locally, and how to release it.
9. **Platform:** add its templates and pin its tag, as in
   [How to deploy](#how-to-deploy).
