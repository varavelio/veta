export default function({ data }) {
  return [
    {
      permalink: "/",
      template: "base.html",
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
