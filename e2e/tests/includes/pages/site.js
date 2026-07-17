export default function({ parse }) {
  const { html } = parse.markdown("<panel>Nested **slot** content.</panel>");

  return [
    {
      permalink: "/",
      template: "base",
      title: "Includes",
      content: parse.renderComponents(html),
    },
  ];
}
