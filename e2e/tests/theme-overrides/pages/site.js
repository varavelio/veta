export default function({ parse }) {
  const home = parse.markdown("# Theme Overlay\n\n<badge>\n\nProject component\n\n</badge>");
  const themeOnly = parse.markdown("Theme template page");

  return [
    {
      permalink: "/",
      template: "base",
      title: "Theme home",
      content: parse.renderComponents(home.html),
    },
    {
      permalink: "/theme-only/",
      template: "theme-only.j2",
      title: "Theme only",
      content: themeOnly.html,
    },
  ];
}
