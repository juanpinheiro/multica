import { NextResponse, type NextRequest } from "next/server";
import { LOCALE_COOKIE } from "@multica/core/i18n";
import {
  MULTICA_LOCALE_HEADER,
  resolveLocaleFromSignals,
} from "./lib/locale-routing";

// Old workspace-scoped route segments that existed before the URL refactor
// (pre-#1131). Any URL with these as the FIRST segment is a legacy URL that
// needs to be rewritten to /{slug}/{route}/... so old bookmarks and deep
// links don't hit 404.
const LEGACY_ROUTE_SEGMENTS = new Set([
  "issues",
  "projects",
  "agents",
  "inbox",
  "my-issues",
  "autopilots",
  "runtimes",
  "skills",
  "settings",
]);

function resolveLocale(req: NextRequest): string {
  return resolveLocaleFromSignals({
    cookieLocale: req.cookies.get(LOCALE_COOKIE)?.value,
    acceptLanguage: req.headers.get("accept-language"),
  });
}

// Forward the resolved locale to RSC layouts via the `x-multica-locale`
// request header. layout.tsx reads it through `await headers()`. The
// `request: { headers }` form is what makes the header land on the upstream
// request — without it the value would only sit on the response.
function nextWithLocale(req: NextRequest): NextResponse {
  const headers = new Headers(req.headers);
  headers.set(MULTICA_LOCALE_HEADER, resolveLocale(req));
  return NextResponse.next({ request: { headers } });
}

// Next.js 16 renamed `middleware` → `proxy`. API surface (NextRequest /
// NextResponse / cookies / matcher) is identical; the only behavioral
// change is the runtime — proxy is forced to nodejs and cannot opt into
// edge.
export function proxy(req: NextRequest) {
  const { pathname } = req.nextUrl;
  const lastSlug = req.cookies.get("last_workspace_slug")?.value;

  const firstSegment = pathname.split("/")[1] ?? "";
  if (LEGACY_ROUTE_SEGMENTS.has(firstSegment)) {
    const url = req.nextUrl.clone();
    url.pathname = lastSlug ? `/${lastSlug}${pathname}` : "/workspaces/new";
    return NextResponse.redirect(url);
  }

  return nextWithLocale(req);
}

export const config = {
  // i18n header must land on every page request, so we use the standard
  // negative-lookahead pattern from Next's i18n guide: skip API routes
  // (Go backend), Next internals, and any path with a file extension
  // (favicons, sw.js, public/* assets).
  matcher: ["/((?!api|_next/static|_next/image|favicon.ico|.*\\.).*)"],
};
