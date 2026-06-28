export default function({ data }) {
  return [
    {
      permalink: "/",
      template: "base",
      title: "home page",
      content: `<callout kind="hero"># ${data.site.title}\n\nTests the **entire build pipeline**.</callout>`,
    },
    {
      permalink: "/docs/intro/",
      template: "base",
      title: "intro guide",
      content: `# Intro\n\nRepo: ${data.github.repo}\n\nTheme: ${data.theme.brand.name}`,
    },
    {
      permalink: "/feed.xml",
      content: `<feed>stars:${data.github.stars}</feed>`,
    },
  ];
}
