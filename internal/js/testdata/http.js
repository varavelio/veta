export default function({ baseURL, httpClient }) {
  const get = httpClient.get(baseURL + "/get?name=veta", {
    headers: { "X-Test": "yes" },
    timeoutMs: 1000,
  });

  const post = httpClient.post(baseURL + "/post", {
    headers: { "X-Trace": ["one", "two"] },
    json: { count: 2, name: "Veta" },
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
      body: JSON.parse(get.body),
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
      body: JSON.parse(JSON.parse(post.body).body),
      contentType: JSON.parse(post.body).contentType,
      ok: post.ok,
      status: post.status,
      traces: JSON.parse(post.body).traces,
    },
    put: {
      body: JSON.parse(put.body).body,
      contentType: JSON.parse(put.body).contentType,
      method: JSON.parse(put.body).method,
      statusText: put.statusText,
    },
  };
}
