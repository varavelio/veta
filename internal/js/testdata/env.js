export default function({ env }) {
  return {
    empty: env.EMPTY,
    keys: Object.keys(env).sort(),
    missingType: typeof env.DOES_NOT_EXIST,
    mode: env.VETA_MODE,
    number: env.VETA_NUMBER,
    numberType: typeof env.VETA_NUMBER,
  };
}
