export default function({ files, parse }) {
  const markdownPage = parse.markdown(files.readFile("content/markdown-page.md"));
  const fileFragment = files.readFile("content/raw-fragment.txt");

  return [
    {
      permalink: "/markdown/",
      template: "page",
      title: markdownPage.frontmatter.title,
      source: "parse.markdown",
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
