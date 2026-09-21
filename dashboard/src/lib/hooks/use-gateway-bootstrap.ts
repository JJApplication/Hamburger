"use client";

import { useEffect } from "react";

import { useGatewayStore } from "@/store/gateway-store";
import { useAuth } from "@/lib/auth/auth-context";

export function useGatewayBootstrap(): void {
  const initialize = useGatewayStore((state) => state.initialize);
  const refresh = useGatewayStore((state) => state.refresh);
  const { ready, token } = useAuth();

  useEffect(() => {
    if (!ready) return;
    if (token) void initialize(token);
    else void refresh();
  }, [initialize, ready, refresh, token]);
}
