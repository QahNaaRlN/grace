import type { SVGProps } from 'react'

// Same ruble and pulse geometry as public/favicon.svg.
export function GraceMark(props: SVGProps<SVGSVGElement>) {
  return (
    <svg viewBox="0 0 64 64" fill="none" stroke="currentColor" strokeWidth="4" strokeLinecap="round" strokeLinejoin="round" {...props}>
      <path d="M49.5 32A23 23 0 1 0 35 48.6" />
      <path d="M18 42V15h9a8 8 0 0 1 0 16H14M14 37h16" />
      <path d="M37 43h6l4-9 5 18 4-9h5" />
    </svg>
  )
}
