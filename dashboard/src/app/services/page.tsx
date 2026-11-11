import { ServiceConfigView } from "@/components/service-config/service-config-view";
import { loadServiceConfig } from "@/lib/service-config/load";

export default function ServicesPage() {
  return <ServiceConfigView config={loadServiceConfig()} />;
}
