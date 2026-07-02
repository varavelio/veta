export default function() {
  return [
    {
      permalink: "/",
      template: "base",
      title: "Theme home",
      content: "# Theme Overlay\n\n<badge>\nProject component\n</badge>",
    },
    {
      permalink: "/theme-only/",
      template: "theme-only.j2",
      title: "Theme only",
      content: "Theme template page",
    },
  ];
}
