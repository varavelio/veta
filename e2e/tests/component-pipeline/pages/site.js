export default function({ files, parse }) {
  const markdownPage = parse.markdown(files.readFile("content/markdown-page.md"));
  const fileFragment = files.readFile("content/raw-fragment.txt");
  const inline = parse.markdown(`<stack name="inline">

<box title="Inline Nested">

Inline **slot** with <ui-layout-blocks-deep-badge label="Inline Deep" />.

</box>

</stack>`);

  return [
    {
      permalink: "/markdown/",
      template: "page",
      title: markdownPage.frontmatter.title,
      source: "parse.markdown",
      content: parse.renderComponents(markdownPage.html),
    },
    {
      permalink: "/file/",
      template: "page",
      title: "File Fragment",
      source: "readFile",
      content: parse.renderComponents(fileFragment),
    },
    {
      permalink: "/inline/",
      template: "page",
      title: "Inline Generator",
      source: "inline-string",
      content: parse.renderComponents(inline.html),
    },
    {
      permalink: "/automatic/",
      template: "page",
      title: "Unprocessed Content",
      source: "unchanged",
      content: "# Unprocessed\n\n<box title=\"Automatic\">Still **raw**.</box>\n\n<p>Ready HTML</p>",
    },
  ];
}
