# Open-Source Release — Shaping

## Source

> I want to open-source this project and I need multiple things done for that. We need great documentation in the form of a README, we need all of the features merged into main, starting from the oldest feature branch and going all the way up to the current v8-mcp-server. I already removed all of the branches that contained tests and/or that aren't relevant. I checked with git log -100 --oneline and it goes linearly all the way back to the Initial commit that started it all. The history is linear so it should be very easy to merge the branches one by one into main, starting from the oldest (v1-extract-article) then move up, always rebasing so as to have a clean and linear history on main. Finally, I want a great website we host on GitHub Pages later.

---

## Frame

### Problem

- All project work (82 commits across V1–V8) lives on feature branches — `main` has only the initial commit
- The empty README makes the project unapproachable — no one can understand what it does or how to use it
- The typst-renderer branch (chromedp → Typst switch) was never given a proper slice designation in the shaping docs
- No public-facing website for discovery, documentation, or demos

### Outcome

- Clean, linear history on `main` with all features merged
- A README that makes the project instantly understandable and usable
- Complete shaping docs that reflect the actual development history (including the Typst renderer switch)
- A polished GitHub Pages site for discoverability and documentation

---

## Requirements (R)

| ID  | Requirement                                                                      | Status    |
|-----|----------------------------------------------------------------------------------|-----------|
| R0  | All feature branches merged into main with clean, linear history                 | Core goal |
| R1  | README covers: what it does, installation, usage (CLI + API + MCP), config       | Must-have |
| R2  | README includes visual examples (screenshots or output samples)                  | Must-have |
| R3  | Merge order preserves the existing ancestry chain                                | Must-have |
| R4  | No merge commits — fast-forward merges only for linear history                   | Must-have |
| R5  | GitHub Pages site with documentation, feature overview, and getting started      | Must-have |
| R6  | MIT LICENSE file added                                                           | Must-have |
| R7  | typst-renderer retroactively documented as a slice in shaping docs               | Must-have |
| R8  | Feature branches deleted after push to GitHub is confirmed                       | Must-have |

---

## Decisions

- **License:** MIT
- **Slice order:** V10 → V9 → V11
- **typst-renderer:** Retroactive slice (V2b) in shaping docs
- **Branch cleanup:** Delete only after push to GitHub confirmed
- **Site generator:** Decide later (during V11 shaping)

---

## Branch Ancestry Chain (Confirmed)

```
main (13e578b) → v1-extract-article → v2-render-pdf → v3-translate → v5-config-queryid → v4-quality → typst-renderer → v6-web-api → v8-mcp-server
```

| Branch            | Commits ahead of main | Incremental commits | Content                                    |
|:------------------|:---------------------:|:-------------------:|:-------------------------------------------|
| v1-extract-article | 20                   | 20                  | GraphQL extraction, Draft.js parser, CLI   |
| v2-render-pdf     | 30                    | 10                  | chromedp PDF renderer, HTML output         |
| v3-translate      | 37                    | 7                   | Ollama translation, batch processing       |
| v5-config-queryid | 51                    | 14                  | YAML config, query ID auto-resolution      |
| v4-quality        | 61                    | 10                  | pdfcpu + text extraction validation        |
| typst-renderer    | 70                    | 9                   | chromedp → Typst switch, dark mode, fonts  |
| v6-web-api        | 76                    | 6                   | HTTP API server, job management            |
| v8-mcp-server     | 82                    | 6                   | MCP stdio server, 4 tools                  |

**Note:** v5 comes before v4 in the chain (developed in that order despite numbering).

---

## Slice Summary

| #   | Slice                           | Mechanism                                                | Demo                                                         |
|-----|---------------------------------|----------------------------------------------------------|--------------------------------------------------------------|
| V10 | Merge to main                   | Fast-forward merges in ancestry order + retroactive docs | "`git log --oneline main` shows clean 82-commit history"     | ✅ Complete |
| V9  | README + docs                   | Write README.md, add MIT LICENSE                         | "Visit repo, instantly understand what this is + how to use" | ✅ Complete |
| V11 | GitHub Pages website            | Static site (TBD generator)                              | "Visit site, see polished docs + feature showcase"           | ⏳ Pending  |

---

## V10: Merge to Main

**Demo:** `git log --oneline main` shows a clean, linear 82-commit history from "Initial commit" through "Add MCP progress notifications".

**Parts:**

| Part  | Mechanism                                                                           |
|-------|-------------------------------------------------------------------------------------|
| V10a  | Fast-forward merge 8 branches in ancestry order                                     |
| V10b  | Retroactive slice doc for typst-renderer (V2b in slices.md)                         |
| V10c  | Push main to GitHub                                                                 |
| V10d  | Delete all feature branches (local + remote) after confirming push                  |

**V10a merge sequence (all fast-forward):**

```bash
git checkout main
git merge --ff-only v1-extract-article
git merge --ff-only v2-render-pdf
git merge --ff-only v3-translate
git merge --ff-only v5-config-queryid
git merge --ff-only v4-quality
git merge --ff-only typst-renderer
git merge --ff-only v6-web-api
git merge --ff-only v8-mcp-server
```

After: `main` tip = `v8-mcp-server` tip (c3c5d6e). 82 commits, linear.

**V10b:** Add a "V2b: Typst Renderer Switch" section to `shaping/slices.md` documenting the 9 commits that replaced chromedp with Typst. Reference the existing plan at `x-article-exporter-plan-4.md`.

---

## V9: README + Docs

**Demo:** Visit the GitHub repo → immediately understand what the tool does, how to install it, and how to use all three modes (CLI, API, MCP).

**Parts:**

| Part | Mechanism                                                                   |
|------|-----------------------------------------------------------------------------|
| V9a  | Add MIT LICENSE file                                                        |
| V9b  | Hero section: project name, one-line description, badges (Go, license)     |
| V9c  | Features overview: CLI, translation, dark mode, API, MCP                   |
| V9d  | Installation: `go install`, prerequisites (Typst, optionally Ollama)       |
| V9e  | Quick start: minimal usage example                                          |
| V9f  | Configuration: config file format, auth cookie setup                       |
| V9g  | Usage sections: CLI flags, API endpoints (link to API.md), MCP setup       |
| V9h  | Visual examples: screenshots of light/dark PDF output                      |
| V9i  | Contributing section (minimal) + link to shaping docs                      |

---

## V11: GitHub Pages Website

**Status:** Needs further shaping after V9 and V10 are done. The README content will inform the site structure.

**Known parts (high-level):**

| Part  | Mechanism                                                    |
|-------|--------------------------------------------------------------|
| V11a  | Choose static site generator (Hugo, Jekyll, Docusaurus, etc) |
| V11b  | Feature showcase with screenshots                            |
| V11c  | Getting started guide                                        |
| V11d  | API reference                                                |
| V11e  | MCP integration guide                                        |
| V11f  | GitHub Actions workflow for Pages deployment                 |

---

## Fit Check (R × V10 + V9 + V11)

| Req | Requirement                                                                      | Status    | V10 + V9 + V11 |
|-----|----------------------------------------------------------------------------------|-----------|:---------------:|
| R0  | All feature branches merged into main with clean, linear history                 | Core goal | ✅              |
| R1  | README covers: what it does, installation, usage (CLI + API + MCP), config       | Must-have | ✅              |
| R2  | README includes visual examples (screenshots or output samples)                  | Must-have | ✅              |
| R3  | Merge order preserves the existing ancestry chain                                | Must-have | ✅              |
| R4  | No merge commits — fast-forward merges only for linear history                   | Must-have | ✅              |
| R5  | GitHub Pages site with documentation, feature overview, and getting started      | Must-have | ✅              |
| R6  | MIT LICENSE file added                                                           | Must-have | ✅              |
| R7  | typst-renderer retroactively documented as a slice in shaping docs               | Must-have | ✅              |
| R8  | Feature branches deleted after push to GitHub is confirmed                       | Must-have | ✅              |
