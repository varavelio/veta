---
title: "Environment And Console"
description: "Use environment variables and JavaScript console output in Veta runtime contexts."
icon: "terminal"
weight: 60
---

# Environment And Console

Veta exposes environment variables and console methods to JavaScript files.

## `env`

`env` is an object containing string environment variables captured from the process running Veta.

```js
export default function({ env }) {
  return {
    mode: env.VETA_MODE || "production",
  };
}
```

Use environment variables for secrets, deployment settings, branch names, or development toggles.

Do not commit secrets into `data/` or `pages/`.

## `console`

The console API is available as a JavaScript global. It is not part of the default export context object.

Supported methods:

```txt
console.debug
console.error
console.info
console.log
console.warn
```

Example:

```js
export default function() {
  console.info("Generating pages");
  return [];
}
```

CLI output is prefixed with the log level:

```txt
[js info] Generating pages
```

Objects and arrays are rendered as JSON-like output.
