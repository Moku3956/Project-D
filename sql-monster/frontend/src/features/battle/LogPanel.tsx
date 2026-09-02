/** LOGタブの中身。 */
export function LogPanel({ logs }: { logs: string[] }) {
  return (
    <div className="h-full overflow-y-auto rounded border border-line bg-deep/60 p-3">
      {logs
        .slice()
        .reverse()
        .map((line, i) => (
          <p key={i} className="mb-2 text-[11px] leading-relaxed text-muted last:mb-0">
            {line}
          </p>
        ))}
    </div>
  )
}
