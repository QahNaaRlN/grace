// "3 операции": the operation count of a category in Russian.
export function operationsText(count: number) {
  const tail = count % 100
  const word =
    tail >= 11 && tail <= 14
      ? 'операций'
      : count % 10 === 1
        ? 'операция'
        : count % 10 >= 2 && count % 10 <= 4
          ? 'операции'
          : 'операций'
  return `${count} ${word}`
}
