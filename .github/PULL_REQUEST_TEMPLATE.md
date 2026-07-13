<!--
Thanks for contributing to Sky Panel. Please read CONTRIBUTING.md if you
haven't. Keep the PR focused on one thing.
-->

## What this does

<!-- A short description of the change and why. Link any related issue. -->

## Component(s)

- [ ] panel-api (Go control plane)
- [ ] web (panel frontend)
- [ ] site (marketing / docs site)

## Checklist

- [ ] The relevant checks pass locally for every component I touched
      (panel-api: `go build ./... && go vet ./... && go test ./...`;
      web: `npm run typecheck && npm test && npm run build`;
      site: `npm run build`).
- [ ] CI is green.
- [ ] `CHANGELOG.md` is updated for user-facing changes (Keep a Changelog format).
- [ ] No secrets, tokens, `.env` files, or the panel database are committed.
- [ ] Commits follow the style (`feat:` / `fix:` prefix with a version suffix,
      e.g. `feat: multi-port servers (v0.24.0)`).
- [ ] If this needs a specific sky-daemon version, it's noted in the changelog.

## Notes for reviewers

<!-- Anything that needs attention: tricky bits, follow-ups, screenshots. -->
