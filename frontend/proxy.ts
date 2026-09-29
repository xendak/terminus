import { NextResponse, type NextRequest } from "next/server";

// Optimistic gate: no session cookie → login. The Go API still decides
// whether the cookie is valid (the app layout asks /api/auth/me).
export function proxy(request: NextRequest) {
  if (request.cookies.has("st_session")) return NextResponse.next();
  const url = request.nextUrl.clone();
  const next = request.nextUrl.pathname + request.nextUrl.search;
  url.pathname = "/login";
  url.search = next && next !== "/" ? `?next=${encodeURIComponent(next)}` : "";
  return NextResponse.redirect(url);
}

export const config = {
  matcher: [
    "/((?!api|_next/static|_next/image|favicon.ico|icon.svg|login|sobre).*)",
  ],
};
