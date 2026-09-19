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
  buildInlineLogDetails,
  type InlineLogSource,
} from '../inline-log-details'

const log: InlineLogSource = {
  quota: 55,
  prompt_tokens: 109,
  completion_tokens: 160,
  content: 'saved content',
  request_id: 'req-1',
  upstream_request_id: 'up-1',
}

describe('inline log details', () => {
  test('prices input, cache read, and output, and splits cache out of the prompt', () => {
    const details = buildInlineLogDetails(
      {
        ...log,
        prompt_tokens: 91,
        completion_tokens: 88,
      },
      {
        model_ratio: 10,
        cache_ratio: 0.1,
        cache_tokens: 82,
        completion_ratio: 5,
        group_ratio: 1,
        request_path: '/v1/chat/completions',
      }
    )

    assert.equal(details.billing?.kind, 'tokens')
    if (details.billing?.kind !== 'tokens') return
    assert.deepEqual(
      details.billing.terms.map((term) => term.labelKey),
      ['Input', 'Cache', 'Output']
    )
    assert.deepEqual(
      details.billing.terms.map((term) => term.detailLabelKey),
      ['Input price', 'Cache read price', 'Output price']
    )
    assert.deepEqual(
      details.billing.terms.map((term) => term.tokens),
      [9, 82, 88]
    )
    assert.ok(Math.abs(details.billing.terms[0].priceUSD - 20) < 1e-9)
    assert.ok(Math.abs(details.billing.terms[1].priceUSD - 2) < 1e-9)
    assert.ok(Math.abs(details.billing.terms[2].priceUSD - 100) < 1e-9)
    assert.equal(details.billing.group?.labelKey, 'Group Ratio')
    assert.equal(details.billing.group?.value, '1')
    assert.equal(details.ratios.length, 0)
    assert.equal(details.requestPath, '/v1/chat/completions')
  })

  test('keeps claude prompt tokens separate from cache', () => {
    const details = buildInlineLogDetails(
      {
        ...log,
        prompt_tokens: 9,
        completion_tokens: 88,
      },
      {
        model_ratio: 10,
        cache_ratio: 0.1,
        cache_tokens: 82,
        completion_ratio: 5,
        claude: true,
      }
    )

    assert.equal(details.billing?.kind, 'tokens')
    if (details.billing?.kind !== 'tokens') return
    assert.equal(details.billing.terms[0]?.tokens, 9)
    assert.equal(details.billing.terms[1]?.tokens, 82)
  })

  test('prefers the user exclusive ratio over the group ratio', () => {
    const details = buildInlineLogDetails(log, {
      model_ratio: 1,
      group_ratio: 1,
      user_group_ratio: 2,
    })

    assert.equal(details.billing?.kind, 'tokens')
    if (details.billing?.kind !== 'tokens') return
    assert.equal(details.billing.group?.labelKey, 'User Exclusive Ratio')
    assert.equal(details.billing.group?.value, '2')
  })

  test('uses per-call billing instead of the token formula', () => {
    const details = buildInlineLogDetails(log, {
      model_price: 0.02,
      model_ratio: 1,
      group_ratio: 1,
    })

    assert.equal(details.billing?.kind, 'per-call')
    assert.equal(
      details.ratios.some((item) => item.labelKey === 'Model ratio'),
      false
    )
  })
})
