import { SecurityConfigView } from "@/components/security-config/security-config-view";
import { loadSecurityConfig } from "@/lib/security-config/load";

export default function SecurityPage() {
  return <SecurityConfigView config={loadSecurityConfig()} />;
}
