export default function({ parse }) {
  const { html } = parse.markdown(`<panel title="Nested component" tone="success">

Component **slot** from page.

</panel>`);

  return [
    {
      permalink: "/",
      template: "pages/article",
      title: "Inheritance",
      extra: "extra from page",
      content: parse.renderComponents(html),
    },
  ];
}
