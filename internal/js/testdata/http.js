export default function({ baseURL, httpClient, parse }) {
  const get = httpClient.get(baseURL + "/get?name=veta", {
    headers: { "X-Test": "yes" },
    timeoutMs: 1000,
  });

  const post = httpClient.post(baseURL + "/post", {
    body: JSON.stringify({ count: 2, name: "Veta" }),
    headers: { "Content-Type": "application/json", "X-Trace": ["one", "two"] },
  });

  const put = httpClient.request("PUT", baseURL + "/echo", {
    body: "plain body",
    headers: { "Content-Type": "text/plain" },
  });

  const deleted = httpClient.delete(baseURL + "/teapot");
  const head = httpClient.head(baseURL + "/head");

  return {
    delete: {
      body: deleted.body,
      ok: deleted.ok,
      status: deleted.status,
    },
    get: {
      body: parse.json(get.body),
      header: get.headers["X-Response"][0],
      ok: get.ok,
      status: get.status,
      urlEndsWith: get.url.endsWith("/get?name=veta"),
    },
    head: {
      body: head.body,
      header: head.headers["X-Head"][0],
      ok: head.ok,
      status: head.status,
    },
    post: {
      body: parse.json(parse.json(post.body).body),
      contentType: parse.json(post.body).contentType,
      ok: post.ok,
      status: post.status,
      traces: parse.json(post.body).traces,
    },
    put: {
      body: parse.json(put.body).body,
      contentType: parse.json(put.body).contentType,
      method: parse.json(put.body).method,
      statusText: put.statusText,
    },
  };
}
