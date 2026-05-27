"use client";

import { useEffect, type ReactNode } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { getApi } from "../api";
import { useAuthStore } from "../auth";
import { configStore } from "../config";
import { workspaceKeys } from "../workspace/queries";
import { createLogger } from "../logger";
import type { User } from "../types";

const logger = createLogger("auth");

export function AuthInitializer({
  children,
  onLogin,
  onLogout,
}: {
  children: ReactNode;
  onLogin?: () => void;
  onLogout?: () => void;
}) {
  const qc = useQueryClient();

  useEffect(() => {
    const api = getApi();

    // Fetch app config (CDN domain, …) in the background — non-blocking.
    api
      .getConfig()
      .then((cfg) => {
        if (cfg.cdn_domain) configStore.getState().setCdnDomain(cfg.cdn_domain);
        configStore.getState().setAuthConfig({
          allowSignup: cfg.allow_signup,
          googleClientId: cfg.google_client_id,
        });
      })
      .catch(() => {
        /* config is optional — legacy file card matching degrades gracefully */
      });

    const onAuthSuccess = (user: User) => {
      onLogin?.();
      useAuthStore.setState({ user, isLoading: false });
    };

    const onAuthFailure = () => {
      onLogout?.();
      useAuthStore.setState({ user: null, isLoading: false });
    };

    // Cookie mode: the HttpOnly cookie is sent automatically by the browser.
    // Seed the workspace list into React Query so the URL-driven layout can
    // resolve the slug without a second fetch.
    Promise.all([api.getMe(), api.listWorkspaces()])
      .then(([user, wsList]) => {
        onAuthSuccess(user);
        qc.setQueryData(workspaceKeys.list(), wsList);
      })
      .catch((err) => {
        logger.error("auth init failed", err);
        onAuthFailure();
      });
  }, []);

  return <>{children}</>;
}
