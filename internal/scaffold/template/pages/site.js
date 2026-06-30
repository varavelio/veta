// Pages are plain JavaScript. Docs: https://veta.varavel.com/pages
// You can access environment, data, local files and remote sources
// using the destructured context.
export default function({ data, files, httpClient }) {
  return [
    {
      permalink: "/",
      template: "base",
      title: "Home",
      content: "<note>A tiny site generated with **Veta**.</note>",
    },
    {
      permalink: "/about/",
      template: "base",
      title: "About",
      content: "This page was generated from `pages/site.js`.",
    },
  ];
}
