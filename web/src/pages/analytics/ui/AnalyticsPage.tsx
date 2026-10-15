import { Disclosure } from '@/shared/ui/disclosure'
import { LedgerScreen } from '@/widgets/ledger-screen'
import { PeriodAnalytics } from '@/widgets/period-analytics'
import { StatisticsPanel } from '@/widgets/statistics-charts'

export function AnalyticsPage() {
  return (
    <LedgerScreen
      title="Статистика"
      description="Посмотрите, куда уходят деньги и как меняются траты."
    >
      {() => (
        <div className="space-y-8">
          <StatisticsPanel />
          <Disclosure
            className="rounded-2xl border bg-card p-5"
            triggerClassName="min-h-11 content-center font-medium"
            title={<>Подробные расчёты остатков и платежей</>}
          >
            <div className="mt-5">
              <PeriodAnalytics />
            </div>
          </Disclosure>
        </div>
      )}
    </LedgerScreen>
  )
}
