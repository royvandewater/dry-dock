# dry-dock

A terminal UI for managing [lazy.vim](https://github.com/folke/lazy.nvim) plugin
updates that enforces a **configurable minimum release age** — so you never
install a plugin version that was published too recently to have been vetted.

## Why

Freshly published commits are the highest-risk moment for a supply-chain
compromise. dry-dock only offers versions that have survived in the wild for at
least a minimum age (14 days by default), giving the community time to notice
and revert bad releases.

## How it works

lazy.vim clones each plugin into `~/.local/share/nvim/lazy/<name>` and pins it to
a branch + commit in `~/.config/nvim/lazy-lock.json`. dry-dock:

1. Reads the lock file to learn each plugin's pinned commit and tracked branch.
2. Fetches each plugin's remote.
3. Treats every commit ahead of the pinned one as a candidate version, using the
   commit date as the release date.
4. Hides candidates younger than the minimum release age.

## Interface

Three panes, driven entirely by the arrow keys:

```
┌ Plugins ┐┌ Versions ────┐┌ Changes ─────────────────┐
│ > blink ││ > abc123 22d ││ current → selected        │
│   tele  ││   def456 25d ││ every commit pulled in by │
│   ...   ││   ...        ││ updating to the selected  │
└─────────┘└──────────────┘└  version, newest first    ┘
```

- **↑ / ↓** — move the highlight in the focused pane.
- **→** — focus the version list (highlights the newest installable version).
- **←** — return focus to the plugin list.
- **enter** — apply the highlighted version: repin it in `lazy-lock.json` and
  let lazy.vim check it out.
- **esc** — dismiss the update status message (restoring the key hints).
- **q** / **ctrl-c** — quit.

Highlighting a plugin lists the versions it can update to, newest first, all
newer than the installed version and all old enough to satisfy the minimum
release age. Highlighting a version shows every change pulled in by updating to
it — the cumulative changelog from the current version through the selected one.
Commits that announce a breaking change (a Conventional Commits `!` marker or
`BREAKING CHANGE`) are tagged **⚠ BREAKING** in the changelog.

### Version constraints

If a plugin's lazy.vim spec pins a version range (e.g. `version = '1.*'`),
dry-dock respects it: the version list shows only the release **tags** that
satisfy the constraint, so a `1.*` plugin never offers `2.x`. dry-dock reads the
constraints straight from lazy.vim (via a headless `nvim`), so they match
exactly what lazy resolves. When newer releases exist outside the range, the
Versions pane notes how many (**⚠ N newer releases outside 1.\***) so you know an
upgrade is available but gated by your own pin. Plugins without a `version`
matcher are tracked commit-by-commit as before.

Pressing **enter** on a version performs the update: dry-dock rewrites
`lazy-lock.json` to pin the chosen commit (matching lazy.vim's own lock format),
then runs `nvim --headless "+Lazy! restore <plugin>" +qa` so lazy.vim performs
the checkout through its own pipeline — installing new dependencies and running
build steps. dry-dock deliberately does not `git checkout` the clone itself: a
raw checkout skips lazy's pipeline and can leave a plugin broken (e.g. a version
bump that pulls in a new dependency).

After the checkout, dry-dock loads the plugin in a headless nvim to confirm it
still works. If nvim reports an error, dry-dock **rolls the plugin back** to the
commit it was on before and reports the failure — so a breaking update can't
leave your editor unusable. On success, the version drops out of the list and
the changelog reflects the plugin's new position.

Only after that success does dry-dock **commit and push** the `lazy-lock.json`
change in the config repo that holds it (e.g. `~/.config/nvim`), with a message
naming the plugin and the version it moved to (`Update telescope.nvim to
abc1234`). A broken-and-rolled-back update is never committed. The commit and
push are best-effort: a config directory that isn't a git repo, or one with no
remote to push to, won't turn a good update into a reported failure.

## Usage

```bash
go build -o dry-dock .
./dry-dock                     # 14-day minimum release age
./dry-dock -min-age-days 30    # stricter
./dry-dock -lock ./lazy-lock.json -install-dir ./plugins
```

## Development

```bash
go test ./...
```
