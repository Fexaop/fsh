export const SESSION_TOKEN_COOKIE = "fsh_session_token";
export const SESSION_COOKIE_MAX_AGE = 30 * 24 * 60 * 60;

export const BACKEND_BASE_URL =
  process.env.NEXT_PUBLIC_BACKEND_URL ?? "http://localhost:8080";

export type GoogleAuthStartResponse = {
  authUrl: string;
  state: string;
};

export type GoogleAuthSessionResponse = {
  status: "pending" | "authenticated" | "expired" | "not_found";
  sessionToken?: string;
  email?: string;
  name?: string;
  picture?: string;
  expiresAt?: number;
};

export function getSessionTokenFromCookieString(
  cookieString: string,
): string | null {
  const prefix = `${SESSION_TOKEN_COOKIE}=`;

  for (const cookie of cookieString.split(";")) {
    const trimmed = cookie.trim();
    if (trimmed.startsWith(prefix)) {
      const token = trimmed.slice(prefix.length);
      return token || null;
    }
  }

  return null;
}

export function hasSessionToken(cookieString: string): boolean {
  return getSessionTokenFromCookieString(cookieString) !== null;
}

export function setSessionTokenCookie(token: string): void {
  document.cookie = `${SESSION_TOKEN_COOKIE}=${token}; path=/; max-age=${SESSION_COOKIE_MAX_AGE}; samesite=lax`;
}

export function clearSessionTokenCookie(): void {
  document.cookie = `${SESSION_TOKEN_COOKIE}=; path=/; max-age=0; samesite=lax`;
}
