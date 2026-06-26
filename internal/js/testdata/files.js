export default function({ files }) {
  return {
    allFiles: files.listFiles("."),
    indexContent: files.readFile("./content/index.md"),
    markdownFiles: files.listFiles("content/**/*.md"),
    textFiles: files.readFiles("content/**/*.txt"),
  };
}
