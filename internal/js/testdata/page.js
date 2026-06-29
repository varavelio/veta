const greeting = "Hello";

export default function(runtime) {
  return {
    title: greeting + ", " + runtime.siteName,
    hasFileAPI: typeof runtime.files.listFiles === "function",
    keys: Object.keys(runtime).sort(),
  };
}
