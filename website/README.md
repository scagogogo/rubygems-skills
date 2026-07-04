# Website

This folder contains the source for the rubygems-skills documentation site, built with [VitePress](https://vitepress.dev).

## Structure

```
website/
├── .vitepress/
│   ├── config.mts       # site config (nav, sidebar, theme, base path)
│   └── theme/           # custom theme (brand color, prompt-card style)
├── docs/
│   ├── index.md         # home page (AI agent prompts front and center)
│   ├── guide/           # what / why / how / install / quick-start / features
│   ├── ai-agents/       # Claude Code & Codex integration + copy-paste prompts
│   ├── api/             # Repository, WriteRepository, models, options, errors
│   ├── cli/             # install, commands, examples
│   └── auto-install/   # cross-platform Ruby installer
├── public/              # favicon
├── package.json
└── package-lock.json
```

## Local development

```bash
cd website
npm install
npm run dev      # http://localhost:5173/rubygems-skills/
```

## Build

```bash
npm run build    # outputs to .vitepress/dist/
npm run preview  # preview the built site locally
```

## Deployment (GitHub Pages)

The site is built and deployed automatically by [`.github/workflows/website.yml`](../.github/workflows/website.yml) on every push to `main` that touches `website/**`.

### One-time setup

After the first workflow run, enable GitHub Pages in the repo settings:

1. Go to **Settings → Pages**.
2. Under **Build and deployment → Source**, select **GitHub Actions**.
3. The `Deploy Website` workflow will publish to `https://<user>.github.io/rubygems-skills/`.

The `base: '/rubygems-skills/'` setting in `config.mts` matches this project-site path. If you move to a custom domain, change `base` to `'/'` and configure the domain in Pages settings.

## Editing

All content is plain Markdown under `docs/`. The nav and sidebar are defined in `config.mts` — add a page by creating the `.md` file and adding an entry to the matching sidebar section.
