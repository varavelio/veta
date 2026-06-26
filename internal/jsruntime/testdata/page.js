const greeting = "Hello";

export default function(runtime) {
  return {
    title: greeting + ", " + runtime.siteName,
    globalAvailable: Veta.siteName === runtime.siteName,
    keys: Object.keys(runtime).sort(),
  };
}
