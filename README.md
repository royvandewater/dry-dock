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
- **q** / **esc** — quit.

Highlighting a plugin lists the versions it can update to, newest first, all
newer than the installed version and all old enough to satisfy the minimum
release age. Highlighting a version shows every change pulled in by updating to
it — the cumulative changelog from the current version through the selected one.

Pressing **enter** on a version performs the update: dry-dock rewrites
`lazy-lock.json` to pin the chosen commit (matching lazy.vim's own lock format),
then runs `nvim --headless "+Lazy! restore <plugin>" +qa` so lazy.vim performs
the checkout through its own pipeline — installing new dependencies and running
build steps. dry-dock deliberately does not `git checkout` the clone itself: a
raw checkout skips lazy's pipeline and can leave a plugin broken (e.g. a version
bump that pulls in a new dependency). Once applied, the version drops out of the
list and the changelog reflects the plugin's new position.

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
