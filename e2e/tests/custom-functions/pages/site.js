export default function({ parse }) {
  const home = parse.markdown(
    "This content comes from the custom functions fixture.\n\n<function-card label=\"Content Badge\" />",
  );
  const docs = parse.markdown(
    "Nested page content for custom function URLs.\n\n<function-card label=\"Content Badge\" />",
  );

  return [
    {
      permalink: "/",
      template: "page",
      title: "Home Page",
      content: home.html,
    },
    {
      permalink: "/docs/intro/",
      template: "page",
      title: "Docs Intro",
      content: docs.html,
    },
  ];
}
