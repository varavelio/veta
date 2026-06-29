export default function() {
  return [
    {
      permalink: "/",
      template: "pages/article",
      title: "Inheritance",
      extra: "extra from page",
      content: "<panel title=\"Nested component\" tone=\"success\">Component **slot** from page.</panel>",
    },
  ];
}
