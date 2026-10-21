export function shortModel(model: string): string {
  return model.replace(/^(claude|gemini)-/, '')
}
