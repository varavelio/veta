export default function({ join, nested, siteName, value }) {
  return {
    functionResult: join(siteName, String(value)),
    globalResult: Veta.join(Veta.siteName, "global"),
    nestedAnswer: nested.answer,
    runtimeKeys: Object.keys(Veta).sort(),
  };
}
