import { StaticConfigView } from "@/components/static-config/static-config-view";
import { loadStaticConfig } from "@/lib/static-config/load";

export default function StaticConfigPage() {
  return <StaticConfigView config={loadStaticConfig()} />;
}
