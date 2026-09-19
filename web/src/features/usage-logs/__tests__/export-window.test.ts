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
import assert from 'node:assert/strict'
import { describe, test } from 'vitest'

import {
  commonLogExportPath,
  isLogExportOutsideRecentWindow,
  LOG_EXPORT_RECENT_SECONDS,
} from '../lib/export-window'

const nowMs = 1_700_000_000_000
const earliest = Math.floor(nowMs / 1000) - LOG_EXPORT_RECENT_SECONDS

describe('common log export window', () => {
  test('lets admins export a range older than 30 days', () => {
    assert.equal(
      isLogExportOutsideRecentWindow({
        isAdmin: true,
        endTimestamp: earliest - 1,
        nowMs,
      }),
      false
    )
  })

  test('rejects a non-admin range that ends before the recent window', () => {
    assert.equal(
      isLogExportOutsideRecentWindow({
        isAdmin: false,
        endTimestamp: earliest - 1,
        nowMs,
      }),
      true
    )
  })

  test('allows a non-admin range that ends on the window boundary', () => {
    assert.equal(
      isLogExportOutsideRecentWindow({
        isAdmin: false,
        endTimestamp: earliest,
        nowMs,
      }),
      false
    )
  })

  test('does not reject a non-admin export when the end is omitted', () => {
    assert.equal(
      isLogExportOutsideRecentWindow({
        isAdmin: false,
        nowMs,
      }),
      false
    )
  })

  test('uses the all-logs endpoint only for the admin view', () => {
    assert.equal(commonLogExportPath(true), '/api/log/export')
    assert.equal(commonLogExportPath(false), '/api/log/self/export')
  })
})
