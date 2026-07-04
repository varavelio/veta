export default function() {
  return [
    {
      permalink: "/",
      template: "page",
      title: "Home Page",
      content: "This content comes from the custom functions fixture.\n\n<function-card label=\"Content Badge\" />",
    },
    {
      permalink: "/docs/intro/",
      template: "page",
      title: "Docs Intro",
      content: "Nested page content for custom function URLs.\n\n<function-card label=\"Content Badge\" />",
    },
  ];
}
