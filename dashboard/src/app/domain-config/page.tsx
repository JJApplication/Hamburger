import { DomainConfigView } from "@/components/domain-config/domain-config-view";
import { loadDomainConfig } from "@/lib/domain-config/load";

export default function DomainConfigPage() {
  return <DomainConfigView config={loadDomainConfig()} />;
}
