function escapeHTML(value) {
  return String(value)
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;");
}

export default function({ files, parse }) {
  const site = parse.json(files.readFile("data/site.json"));
  const navigation = parse.yaml(files.readFile("data/navigation.yaml"));
  const theme = parse.toml(files.readFile("data/theme.toml"));
  const yamlPost = parse.markdown(files.readFile("content/articles/yaml.md"));
  const tomlPost = parse.markdown(files.readFile("content/articles/toml.md"));
  const plainPost = parse.markdown(files.readFile("content/snippets/plain.md"));
  const note = files.readFile("content/plain.txt").trim();
  const markdownFiles = files.listFiles("content/**/*.md");
  const permalinks = markdownFiles.map((file) => files.toPermalink(file, { stripPrefix: "content" }));

  return [
    {
      permalink: "/",
      content: `<!doctype html>
<html lang="en">
<body>
  <h1>${escapeHTML(site.title)}</h1>
  <p data-nav="${escapeHTML(navigation.main[0].label)}">${escapeHTML(navigation.main[0].href)}</p>
  <p data-theme="${escapeHTML(theme.colors.primary)}">${escapeHTML(theme.name)}</p>
  <article data-source="yaml">
    <h2>${escapeHTML(yamlPost.frontmatter.title)}</h2>
    <p>${escapeHTML(yamlPost.frontmatter.tags.join(","))}</p>
    <pre>${escapeHTML(yamlPost.content)}</pre>
  </article>
  <article data-source="toml">
    <h2>${escapeHTML(tomlPost.frontmatter.title)}</h2>
    <p>${escapeHTML(tomlPost.frontmatter.meta.author)}</p>
    <pre>${escapeHTML(tomlPost.content)}</pre>
  </article>
  <p data-plain-path="content/snippets/plain.md">${escapeHTML(plainPost.content.trim())}</p>
  <p data-note="${escapeHTML(note)}">${escapeHTML(markdownFiles.join(";"))}</p>
  <p data-permalinks="${escapeHTML(permalinks.join(";"))}"></p>
</body>
</html>
`,
    },
    {
      permalink: "/files.json",
      content: JSON.stringify(
        {
          files: markdownFiles,
          permalinks,
          tomlTitle: tomlPost.frontmatter.title,
          yamlTitle: yamlPost.frontmatter.title,
        },
        null,
        2,
      ),
    },
  ];
}
