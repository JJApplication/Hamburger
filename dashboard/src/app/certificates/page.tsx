import { CertificateView } from "@/components/certificates/certificate-view";
import { loadCertificates } from "@/lib/certificates/load";

export default function CertificatesPage() {
  return <CertificateView model={loadCertificates()} />;
}
