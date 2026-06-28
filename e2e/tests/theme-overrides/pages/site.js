export default function() {
  return [
    {
      permalink: "/",
      layout: "templates/base",
      title: "Theme home",
      content: "# Theme Overlay\n\n<badge>\nProject component\n</badge>",
    },
    {
      permalink: "/theme-only/",
      layout: "templates/theme-only",
      title: "Theme only",
      content: "Theme template page",
    },
  ];
}
