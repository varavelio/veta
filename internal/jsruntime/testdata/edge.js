const samples = [
  "import value from elsewhere",
  "export const hidden = true",
];

function thenableButNotPromise() {
  return { then: "not a function" };
}

export default function() {
  return {
    bool: true,
    nullValue: null,
    number: 12.5,
    promiseType: typeof Promise,
    samples,
    thenableButNotPromise: thenableButNotPromise(),
  };
}
