export default function({ data, parse }) {
  const home = parse.markdown(`# ${data.site.name}`);
  const docs = parse.markdown("Nested config works.");

  return [
    {
      permalink: "/",
      template: "base.j2",
      title: "Home",
      content: home.html,
    },
    {
      permalink: "docs/getting-started",
      template: "docs/page",
      title: "Docs",
      content: docs.html,
    },
  ];
}
