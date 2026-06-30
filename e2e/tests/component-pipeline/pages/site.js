export default function({ files }) {
  const markdownPage = files.readMarkdownFile("content/markdown-page.md");
  const fileFragment = files.readFile("content/raw-fragment.txt");

  return [
    {
      permalink: "/markdown/",
      template: "page",
      title: markdownPage.frontmatter.title,
      source: "readMarkdownFile",
      content: markdownPage.content,
    },
    {
      permalink: "/file/",
      template: "page",
      title: "File Fragment",
      source: "readFile",
      content: fileFragment,
    },
    {
      permalink: "/inline/",
      template: "page",
      title: "Inline Generator",
      source: "inline-string",
      content: `<stack name="inline">
<box title="Inline Nested">
Inline **slot** with <ui-layout-blocks-deep-badge label="Inline Deep" />.
</box>
</stack>`,
    },
  ];
}
