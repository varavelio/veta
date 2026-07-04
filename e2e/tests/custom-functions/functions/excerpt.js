export default function(context, value, length) {
  const { page } = context;
  console.log("excerpt", page.permalink);
  if (context.console !== undefined) {
    throw new Error("console should be global, not on page");
  }

  return String(value).slice(0, Number(length));
}
