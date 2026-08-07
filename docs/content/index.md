---
title: "Veta"
description: "A static site generator that combines JavaScript page generators, templates, data, components, filters, Markdown, themes, and embedded Tailwind CSS into a single binary."
---

<vara-header
container="lg"
show_title="false"
links="Documentation|/docs/getting-started/,Guides|/docs/guides/,API|/docs/api/,GitHub|https://github.com/varavelio/veta"
cta_label="Get Started"
cta_href="/docs/getting-started/"
/>

<vara-hero
container="lg"
eyebrow="Veta Static Site Generator"
title="Build static sites with no complex tooling"
description="Veta turns project files and APIs into a static output directory by using simple JavaScript page generators, Pongo templates, components, filters, themes, and Tailwind CSS. All in one binary with no frontend toolchain to maintain."
primary_label="Explore Documentation"
primary_href="/docs/getting-started/"
secondary_label="Install Veta"
secondary_href="/docs/installation/"
panel_icon="panels-top-left"
panel_title="Everything in one binary"
panel_description="The JavaScript runtime, template engine, Markdown renderer, and Tailwind CSS all ship inside the same binary. Nothing else to install."
caption="No Node.js, no Tailwind to install, the CLI is all you need."
item_1="JavaScript page generators with an explicit API"
item_2="Pongo templates, components, filters, and functions"
item_3="JSON, YAML, TOML, and JavaScript data files"
/>

<vara-features
container="lg"
title="What Veta gives you"
description="A focused set of building blocks that compose into any static site."
columns="3"
item_1_icon="file-code"
item_1_title="Page Generators"
item_1_description="Flat JavaScript files declare exactly which pages to build and which templates they use. No hidden routes."
item_1_badge="pages/"
item_2_icon="layout-template"
item_2_title="Pongo Templates"
item_2_description="Jinja-style templates with inheritance, includes, macros, filters, and a small, explicit context."
item_2_badge="templates/"
item_3_icon="database"
item_3_title="Structured Data"
item_3_description="Global data files in JSON, YAML, TOML, or JavaScript, plus on-demand loading from templates."
item_3_badge="data/"
item_4_icon="blocks"
item_4_title="Components"
item_4_description="Reusable component templates resolved explicitly with props and slots, without a frontend framework."
item_4_badge="components/"
item_5_icon="file-text"
item_5_title="Content From Any Source"
item_5_description="Read any project file or fetch any HTTP API (Markdown, JSON, YAML, or any format) and shape it into pages from JavaScript."
item_5_badge="any source"
item_6_icon="palette"
item_6_title="Themes And Tailwind CSS"
item_6_description="Compose local or remote themes with your project and style the result with the Tailwind CSS engine built into the binary, nothing to install."
item_6_badge="theme"
/>

<vara-content-split
container="lg"
eyebrow="How it works"
title="Explicit, synchronous, and predictable"
description="Veta separates what you write from how it looks. Your project declares pages, writes content, and configures a few settings; Veta does the rest at build time."
item_1="Page generators return page objects with permalinks and templates"
item_2="Markdown and components are rendered explicitly when you ask for them"
item_3="Data is loaded globally before generation and shared everywhere"
item_4="The output is plain static files you can deploy anywhere"
panel_icon="workflow"
panel_title="Batteries included"
panel_description="No Node.js runtime, no Tailwind install, no bundler, no configuration drift. Download the CLI, add a theme and build."
heading_level="3"
/>

<vara-faq container="lg" title="Common questions" description="A few things people ask before getting started." open_first="true" heading_level="3">
<vara-faq-item id="what-is-veta" question="What exactly is Veta?">
Veta is a static site generator distributed as a single CLI binary. It uses JavaScript page generators, Pongo templates, structured data, components, filters, Markdown, themes, and embedded Tailwind CSS to produce a static output directory.
</vara-faq-item>
<vara-faq-item id="do-i-need-nodejs" question="Do I need Node.js or a frontend build?">
No. Veta runs its own embedded JavaScript runtime and ships the Tailwind CSS engine inside the binary. There's no Node.js, no package manager, and no bundler to install - one download is everything your project needs.
</vara-faq-item>
<vara-faq-item id="tailwind-included" question="Do I need to install Tailwind CSS?">
No. Tailwind CSS is built into the Veta binary. You point Veta at an input stylesheet and it compiles and minifies your CSS on every build, so there's nothing to install or keep in sync. For a typical project, the CLI is the only requirement. You'd only reach for external tooling if you deliberately go beyond what Veta ships - for example, building a very complex theme with its own custom pipeline.
</vara-faq-item>
<vara-faq-item id="markdown-automatic" question="Is Markdown rendered automatically?">
No. Markdown and component rendering are explicit operations. Page generators call <code>parse.markdown</code> and <code>parse.renderComponents</code> when they need them, which keeps output predictable.
</vara-faq-item>
<vara-faq-item id="themes" question="Can I share templates across projects?">
Yes. Veta supports local and remote themes that provide templates, components, filters, functions, data, and public assets - all running from the same binary with no extra dependencies. Project files always override theme files.
</vara-faq-item>
<vara-faq-item id="deployment" question="Where can I deploy the output?">
Anywhere that serves static files. The build output is a plain directory of HTML, assets, and generated files, so it works with every static hosting provider.
</vara-faq-item>
</vara-faq>

<vara-cta
container="lg"
title="Ready to build?"
description="Install Veta, create a project, and write your first page in minutes."
primary_label="Get Started"
primary_href="/docs/getting-started/"
secondary_label="Installation"
secondary_href="/docs/installation"
/>

<vara-footer
container="lg"
copyright="A Varavel project &copy; %year%"
links="Documentation|/docs,GitHub|https://github.com/varavelio/veta"
github_href="https://github.com/varavelio/veta"
github_label="Veta on GitHub"
/>
