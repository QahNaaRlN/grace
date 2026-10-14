// SPEC-0047: currencies offered in forms; RUB first and default. Go still
// validates any ISO 4217 code, so a stored code outside this list stays valid.
export const DEFAULT_CURRENCY = 'RUB'

export const currencies: [code: string, name: string][] = [
  ['RUB', 'Российский рубль'],
  ['USD', 'Доллар США'],
  ['EUR', 'Евро'],
  ['CNY', 'Китайский юань'],
  ['KZT', 'Казахстанский тенге'],
  ['BYN', 'Белорусский рубль'],
  ['TRY', 'Турецкая лира'],
  ['AED', 'Дирхам ОАЭ'],
  ['GBP', 'Фунт стерлингов'],
]
