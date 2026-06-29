// Pages are plain JavaScript. Docs: https://veta.varavel.com/pages
// You can read local files with Veta.files or request data with Veta.httpClient.
export default function({ data }) {
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
