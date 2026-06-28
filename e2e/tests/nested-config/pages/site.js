export default function({ data }) {
  return [
    {
      permalink: "/",
      layout: "templates/base",
      title: "Home",
      content: `# ${data.site.name}`,
    },
    {
      permalink: "docs/getting-started",
      layout: "templates/base",
      title: "Docs",
      content: "Nested config works.",
    },
  ];
}
