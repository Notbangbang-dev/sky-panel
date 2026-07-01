export function bytesToBase64(bytes: Uint8Array): string {
  let binary = "";
  for (const b of bytes) binary += String.fromCharCode(b);
  return btoa(binary);
}

export function base64ToBytes(base64: string): Uint8Array {
  const binary = atob(base64);
  const bytes = new Uint8Array(binary.length);
  for (let i = 0; i < binary.length; i++) bytes[i] = binary.charCodeAt(i);
  return bytes;
}

export function encodeUtf8Base64(text: string): string {
  return bytesToBase64(new TextEncoder().encode(text));
}

export function decodeUtf8Base64(base64: string): string {
  return new TextDecoder().decode(base64ToBytes(base64));
}
