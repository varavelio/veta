export default function({ data }) {
  return [
    {
      permalink: "/",
      layout: "templates/base",
      title: "home page",
      content: `<callout kind="hero"># ${data.site.title}\n\nTests the **entire build pipeline**.</callout>`,
    },
    {
      permalink: "/docs/intro/",
      layout: "templates/base",
      title: "intro guide",
      content: `# Intro\n\nRepo: ${data.github.repo}\n\nTheme: ${data.theme.brand.name}`,
    },
    {
      permalink: "/feed.xml",
      layout: "templates/plain",
      content: `<feed>stars:${data.github.stars}</feed>`,
    },
  ];
}
