export default function({ listFiles, readFile, readFiles }) {
  return {
    allFiles: listFiles("."),
    indexContent: readFile("./content/index.md"),
    markdownFiles: listFiles("content/**/*.md"),
    textFiles: readFiles("content/**/*.txt"),
  };
}
