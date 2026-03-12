"use client";

import { useEffect } from "react";

import { useGatewayStore } from "@/store/gateway-store";

export function useGatewayBootstrap(): void {
  const initialize = useGatewayStore((state) => state.initialize);

  useEffect(() => {
    void initialize();
  }, [initialize]);
}
