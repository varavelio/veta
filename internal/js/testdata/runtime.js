export default function(runtime) {
  const { join, nested, siteName, value } = runtime;

  return {
    contextResult: join(siteName, "context"),
    functionResult: join(siteName, String(value)),
    nestedAnswer: nested.answer,
    runtimeKeys: Object.keys(runtime).sort(),
  };
}
