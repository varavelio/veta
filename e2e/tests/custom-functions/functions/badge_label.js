export default function({ files, parse }, prefix) {
  const badge = parse.yaml(files.readFile("data/badge.yaml"));
  return `${prefix}: ${badge.label}`;
}
