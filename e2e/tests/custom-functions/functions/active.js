export default function({ page }, href) {
  return page.permalink === href ? "active" : "";
}
