export default function({ parse }, content) {
  return parse.renderComponents(String(content));
}
