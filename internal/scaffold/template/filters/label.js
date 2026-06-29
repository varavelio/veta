// Filters transform values inside templates. Docs: https://veta.varavel.com/filters
export default function(context, input) {
  return `Site: ${String(input)}`;
}
