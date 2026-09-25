export function money(n?: number | null) {
  const v = Number(n ?? 0)
  if (!Number.isFinite(v)) return '—'
  return v.toLocaleString(undefined, {
    minimumFractionDigits: 2,
    maximumFractionDigits: 4,
  })
}

export function displayName(user: {
  username?: string
  Username?: string
  display_name?: string
  DisplayName?: string
} | null) {
  if (!user) return 'Root'
  return (
    user.display_name ||
    user.DisplayName ||
    user.username ||
    user.Username ||
    'Root'
  )
}
