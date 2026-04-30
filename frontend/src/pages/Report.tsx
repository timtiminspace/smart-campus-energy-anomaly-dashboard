import Layout from "@/components/Layout";
import AnonymousReportForm from "@/components/AnonymousReportForm";
import { useEnergyData } from "@/hooks/useEnergyData";

export default function Report() {
  const { buildings, submitReport } = useEnergyData();

  return (
    <Layout>
      <div className="max-w-2xl mx-auto">
        <AnonymousReportForm buildings={buildings} onSubmit={submitReport} />
      </div>
    </Layout>
  );
}
