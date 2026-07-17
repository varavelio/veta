export default function({ parse }) {
  const { html } = parse.markdown("<badge>\n\nComponent **slot**.\n\n</badge>");

  return [
    {
      permalink: "/",
      template: "base",
      content: parse.renderComponents(html),
    },
  ];
}
