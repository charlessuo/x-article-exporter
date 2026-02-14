---
title: X Article Exporter
layout: hextra-home
---

{{< hextra/hero-badge >}}
  Open Source &middot; MIT License
{{< /hextra/hero-badge >}}

<div class="hx-mt-6 hx-mb-6">
{{< hextra/hero-headline >}}
  Export X articles as&nbsp;high&#8209;quality&nbsp;PDFs
{{< /hextra/hero-headline >}}
</div>

<div class="hx-mb-12">
{{< hextra/hero-subtitle >}}
  Fetch, translate, and render X (Twitter) articles as shareable PDFs &mdash; from CLI, HTTP API, or Claude Code.
{{< /hextra/hero-subtitle >}}
</div>

<div class="hx-mb-6">
{{< hextra/hero-button text="Get Started" link="docs/installation" >}}
{{< hextra/hero-button text="View on GitHub" link="https://github.com/annismckenzie/x-article-exporter" style="margin-left: 8px;" >}}
</div>

<div class="hx-mt-6">

| | |
|:---:|:---:|
| ![Light mode](images/example-light.png) | ![Dark mode](images/example-dark.png) |
| Light mode | Dark mode |

</div>

{{< hextra/feature-grid >}}
  {{< hextra/feature-card
    title="CLI"
    subtitle="Export any X article URL to PDF from the terminal. One command, one PDF."
    icon="terminal"
  >}}
  {{< hextra/feature-card
    title="Translation"
    subtitle="Translate articles via local Ollama + translategemma. 55+ languages, fully offline."
    icon="translate"
  >}}
  {{< hextra/feature-card
    title="Dark Mode"
    subtitle="Black-background rendering matching X's dark theme. Looks great on any screen."
    icon="moon"
  >}}
  {{< hextra/feature-card
    title="HTTP API"
    subtitle="Self-hosted server with async job processing, rate limiting, and bearer auth."
    icon="server"
  >}}
  {{< hextra/feature-card
    title="MCP Server"
    subtitle="Use as a Claude Code tool to export articles directly from conversation."
    icon="chip"
  >}}
  {{< hextra/feature-card
    title="Quality Validation"
    subtitle="Structural PDF checks via pdfcpu + text extraction verification. Trust the output."
    icon="shield-check"
  >}}
{{< /hextra/feature-grid >}}
