export default function({ data, parse }) {
  const home = parse.markdown(`<callout kind="hero">

# ${data.site.title}

Tests the **entire build pipeline**.

</callout>`);
  const docs = parse.markdown(`# Intro\n\nRepo: ${data.github.repo}\n\nTheme: ${data.theme.brand.name}`);

  return [
    {
      permalink: "/",
      template: "base",
      title: "home page",
      content: parse.renderComponents(home.html),
    },
    {
      permalink: "/docs/intro/",
      template: "base",
      title: "intro guide",
      content: docs.html,
    },
    {
      permalink: "/feed.xml",
      content: `<feed>stars:${data.github.stars}</feed>`,
    },
  ];
}
