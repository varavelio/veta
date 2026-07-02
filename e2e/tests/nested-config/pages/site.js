export default function({ data }) {
  return [
    {
      permalink: "/",
      template: "base.j2",
      title: "Home",
      content: `# ${data.site.name}`,
    },
    {
      permalink: "docs/getting-started",
      template: "docs/page",
      title: "Docs",
      content: "Nested config works.",
    },
  ];
}
