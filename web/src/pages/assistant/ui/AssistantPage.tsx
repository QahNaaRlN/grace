import { AIAnalysis } from '@/features/ai-analysis'
import { AskAssistant, queryAssistant, type SubmitAssistant } from '@/features/ask-assistant'
import { ReceiptPhoto } from '@/features/recognize-receipt-photo'
import { PageHeader } from '@/shared/ui/page-header'
import { AIGuide } from './AIGuide'

export function AssistantPage({
  submitAssistant = queryAssistant,
}: {
  submitAssistant?: SubmitAssistant
}) {
  return (
    <section className="space-y-6">
      <PageHeader
        title="Ассистент"
        description="Спросите о своих деньгах. Ответы опираются на проверяемые расчёты Grace."
      />
      <AIAnalysis />
      <ReceiptPhoto />
      <AIGuide />
      <AskAssistant submitAssistant={submitAssistant} />
    </section>
  )
}
