export default function({ files, parse }) {
  const markdownFiles = files.listFiles("content/**/*.md");

  return {
    allFiles: files.listFiles("."),
    indexContent: files.readFile("./content/index.md"),
    json: parse.json(files.readFile("data/site.json")),
    markdownFiles,
    markdownPages: markdownFiles.map((path) => {
      const file = parse.markdown(files.readFile(path));
      return {
        content: file.content,
        frontmatter: file.frontmatter,
        path,
        permalink: files.toPermalink(path, { stripPrefix: "content" }),
      };
    }),
    textContent: files.readFile("content/drafts/ignore.txt"),
    toml: parse.toml(files.readFile("data/theme.toml")),
    yaml: parse.yaml(files.readFile("data/navigation.yaml")),
  };
}
