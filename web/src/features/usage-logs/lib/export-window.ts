/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

/** Must stay aligned with model.LogExportRecentWindow. */
export const LOG_EXPORT_RECENT_SECONDS = 30 * 24 * 60 * 60

export function commonLogExportPath(isAdminView: boolean): string {
  return isAdminView ? '/api/log/export' : '/api/log/self/export'
}

/**
 * Non-admins cannot export a range that ends before the recent window.
 * Admins and root users are not limited. A missing end is not a rejection;
 * the server clamps an open start instead.
 */
export function isLogExportOutsideRecentWindow(input: {
  isAdmin: boolean
  endTimestamp?: number
  nowMs: number
}): boolean {
  if (input.isAdmin) return false
  if (input.endTimestamp == null || !Number.isFinite(input.endTimestamp)) {
    return false
  }
  const earliest = Math.floor(input.nowMs / 1000) - LOG_EXPORT_RECENT_SECONDS
  return input.endTimestamp < earliest
}
