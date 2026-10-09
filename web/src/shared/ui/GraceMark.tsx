import type { SVGProps } from 'react'

// Same currency-neutral coin geometry as public/favicon.svg.
export function GraceMark(props: SVGProps<SVGSVGElement>) {
  return (
    <svg viewBox="0 0 64 64" fill="none" stroke="currentColor" strokeWidth="5" strokeLinecap="round" strokeLinejoin="round" {...props}>
      <circle cx="32" cy="32" r="25" />
      <path d="M24 25h16M24 39h16" />
    </svg>
  )
}
