export default function({ files }) {
  const markdownFiles = files.listFiles("content/**/*.md");

  return {
    allFiles: files.listFiles("."),
    indexContent: files.readFile("./content/index.md"),
    json: files.readJsonFile("data/site.json"),
    markdownFiles,
    markdownPages: markdownFiles.map((path) => {
      const file = files.readMarkdownFile(path);
      return {
        content: file.content,
        frontmatter: file.frontmatter,
        path: file.path,
        permalink: files.toPermalink(path, { basePath: "content" }),
      };
    }),
    textContent: files.readFile("content/drafts/ignore.txt"),
    toml: files.readTomlFile("data/theme.toml"),
    yaml: files.readYamlFile("data/navigation.yaml"),
  };
}
