import { Component, type ReactNode } from 'react'

import type { AssistantClarify, AssistantResult } from '@/shared/api'
import { CompletedResult } from './CompletedResult'
import { Confidence, MalformedResult, Notice } from './result-parts'

const clarificationText: Record<AssistantClarify['clarificationCode'], string> = {
  time_precision_required:
    'В истории есть операции с неизвестным временем. Выберите полный календарный период вместо расчёта после точного момента.',
  low_confidence: 'Сформулируйте один конкретный вопрос о ваших финансовых данных.',
  clarification_required: 'Уточните, какие именно финансовые данные вы хотите увидеть.',
  composition_mismatch: 'Не удалось безопасно подобрать параметры. Уточните запрос.',
  arguments_required: 'Для этого расчёта не хватает обязательных параметров.',
  income_category_required:
    'Названную категорию дохода пока нельзя безопасно сопоставить. Спросите о последнем подтверждённом доходе без названия категории.',
  no_income_anchor: 'Подтверждённый доход, от которого можно построить период, не найден.',
}

type RequiredArgument = NonNullable<AssistantClarify['requiredArguments']>[number]

const argumentLabels: Record<RequiredArgument, string> = {
  from: 'дата начала',
  to: 'дата окончания',
  through: 'горизонт обязательств',
  asOf: 'дата состояния',
  limit: 'количество чеков',
  categoryId: 'категория дохода',
}

export class ResultBoundary extends Component<
  { result: AssistantResult; children: ReactNode },
  { failed: boolean }
> {
  state = { failed: false }

  static getDerivedStateFromError() {
    return { failed: true }
  }

  componentDidUpdate(previous: Readonly<{ result: AssistantResult; children: ReactNode }>) {
    if (previous.result !== this.props.result && this.state.failed) {
      this.setState({ failed: false })
    }
  }

  render() {
    return this.state.failed ? <MalformedResult /> : this.props.children
  }
}

export function AssistantResultView({ result }: { result: AssistantResult }) {
  switch (result.status) {
    case 'completed':
      return <CompletedResult result={result} />
    case 'clarify': {
      const required = result.requiredArguments ?? []
      return (
        <Notice
          title="Нужно уточнение"
          text={clarificationText[result.clarificationCode] ?? 'Уточните запрос и повторите.'}
        >
          {required.length > 0 && (
            <p className="text-sm">
              Нужно указать: {required.map((argument) => argumentLabels[argument]).join(', ')}.
            </p>
          )}
          <Confidence value={result.confidence} route={result.route} />
        </Notice>
      )
    }
    case 'unsupported':
      return (
        <Notice
          title="Запрос пока не поддерживается"
          text="Этот ассистент только читает поддерживаемые финансовые данные и ничего не изменяет."
        >
          <Confidence value={result.confidence} route={result.route} />
        </Notice>
      )
    case 'unavailable':
      return result.errorCode === 'dependency_unavailable' ? (
        <Notice
          title="Финансовые данные временно недоступны"
          text="Grace не смог получить данные. Повторите запрос позже."
        />
      ) : (
        <Notice
          title="Ассистент недоступен"
          text="Он выключен или временно не отвечает. Остальные функции Grace продолжают работать."
        />
      )
    default:
      return <MalformedResult />
  }
}
