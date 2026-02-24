import type { NextRequest } from "next/server";
import { NextResponse } from "next/server";

import { SESSION_TOKEN_COOKIE } from "@/lib/auth-constants";

export function middleware(request: NextRequest) {
  const isAuthenticated = Boolean(
    request.cookies.get(SESSION_TOKEN_COOKIE)?.value,
  );
  const { pathname } = request.nextUrl;

  if (
    (pathname.startsWith("/dashboard") || pathname === "/setting") &&
    !isAuthenticated
  ) {
    return NextResponse.redirect(new URL("/login", request.url));
  }

  if (pathname === "/login" && isAuthenticated) {
    return NextResponse.redirect(new URL("/dashboard", request.url));
  }

  return NextResponse.next();
}

export const config = {
  matcher: ["/dashboard/:path*", "/setting", "/login"],
};
