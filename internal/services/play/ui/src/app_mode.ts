// AppMode captures the remaining browser surfaces after isolated component work
// moved out of the bundled SPA and into Storybook.
export type AppMode =
  | { kind: "root-placeholder" }
  | { kind: "runtime-placeholder"; campaignId: string }
  | { kind: "unsupported"; path: string };

// AppResolution keeps startup routing explicit even though the SPA no longer
// performs preview-route redirects.
export type AppResolution = { kind: "render"; mode: AppMode };

type LocationLike = Pick<Location, "pathname" | "search">;

// resolveAppLocation keeps the SPA focused on runtime placeholders and fails
// closed for retired preview paths.
export function resolveAppLocation(location: LocationLike): AppResolution {
  const pathname = location.pathname.trim() || "/";

  if (pathname === "/") {
    return {
      kind: "render",
      mode: { kind: "root-placeholder" },
    };
  }

  const campaignMatch = pathname.match(/^\/campaigns\/([^/?#]+)/);
  if (campaignMatch?.[1]) {
    return {
      kind: "render",
      mode: {
        kind: "runtime-placeholder",
        campaignId: decodeURIComponent(campaignMatch[1]),
      },
    };
  }

  return {
    kind: "render",
    mode: {
      kind: "unsupported",
      path: pathname,
    },
  };
}

// canonicalizeWindowLocation resolves the current browser location once at
// startup so the SPA renders from one explicit shell mode.
export function canonicalizeWindowLocation(location: LocationLike = window.location): AppMode {
  return resolveAppLocation(location).mode;
}
