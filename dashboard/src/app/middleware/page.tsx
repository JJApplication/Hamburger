import { MiddlewareConfigView } from "@/components/middleware-config/middleware-config-view";
import { loadMiddlewareConfig } from "@/lib/middleware-config/load";

export default function MiddlewarePage() {
  return <MiddlewareConfigView config={loadMiddlewareConfig()} />;
}
