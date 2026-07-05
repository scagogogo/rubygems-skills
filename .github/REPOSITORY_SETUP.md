# GitHub Repository Metadata

This file documents the GitHub repository settings currently applied to [scagogogo/rubygems-skills](https://github.com/scagogogo/rubygems-skills).

## Description (one-line)

```
A production-ready Go SDK for the RubyGems.org API — built for AI agents. Docs: https://scagogogo.github.io/rubygems-skills/
```

## Homepage (website)

```
https://scagogogo.github.io/rubygems-skills/
```

The homepage points to the VitePress documentation site deployed via GitHub Pages from the `website/` directory (workflow: `.github/workflows/website.yml`).

## Topics (19 tags)

```
rubygems
rubygems-api
rubygems-client
golang
go-sdk
ruby-gems
gem
api-client
cli
cache
retry
backoff
batch-operations
mirror
ruby-china
private-gem-server
cobra
ai-agent
claude-code
```

## How to Apply

These settings are already applied. To re-apply (requires `gh` CLI and admin permissions):

```bash
# Description
gh repo edit scagogogo/rubygems-skills \
  --description "A production-ready Go SDK for the RubyGems.org API — built for AI agents. Docs: https://scagogogo.github.io/rubygems-skills/"

# Homepage (GitHub Pages)
gh repo edit scagogogo/rubygems-skills \
  --homepage "https://scagogogo.github.io/rubygems-skills/"

# Topics
gh repo edit scagogogo/rubygems-skills \
  --add-topic rubygems,rubygems-api,rubygems-client,golang,go-sdk,ruby-gems,gem,api-client,cli,cache,retry,backoff,batch-operations,mirror,ruby-china,private-gem-server,cobra,ai-agent,claude-code
```

Or via GitHub UI: ⚙️ **Settings → General** on the repository page.

## Social Preview Image

Consider adding a social preview image at `website/public/social-preview.png` (1280×640 recommended).
