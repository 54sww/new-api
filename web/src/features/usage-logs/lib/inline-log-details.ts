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
import type { UsageLog } from '../data/schema'
import type { LogOtherData } from '../types'
import { getTieredBillingSummary } from './format'
import { isPerCallBilling } from './utils'

export type InlineLogSource = Pick<
  UsageLog,
  | 'quota'
  | 'prompt_tokens'
  | 'completion_tokens'
  | 'content'
  | 'request_id'
  | 'upstream_request_id'
>

export type RatioChip = {
  labelKey: string
  value: string
}

export type InlineGroupRatio = {
  labelKey: 'Group Ratio' | 'User Exclusive Ratio'
  value: string
}

export type FormulaTerm = {
  labelKey: string
  detailLabelKey: string
  tokens: number
  priceUSD: number
}

export type TokenBilling = {
  kind: 'tokens'
  terms: FormulaTerm[]
  group: InlineGroupRatio | null
}

export type PerCallBillingDetails = {
  kind: 'per-call'
  priceUSD: number
  group: InlineGroupRatio | null
}

export type TieredBillingDetails = {
  kind: 'tiered'
  prices: Array<{ labelKey: string; priceUSD: number }>
  group: InlineGroupRatio | null
}

export type InlineBilling =
  | TokenBilling
  | PerCallBillingDetails
  | TieredBillingDetails

export type InlineLogDetails = {
  ratios: RatioChip[]
  billing: InlineBilling | null
  requestId: string
  upstreamRequestId: string
  requestPath: string
  content: string
}

function formatExactNumber(value: number): string {
  const fixed = value.toFixed(12)
  const dot = fixed.indexOf('.')
  if (dot < 0) return fixed
  let end = fixed.length
  while (end > dot + 1 && fixed[end - 1] === '0') end -= 1
  if (fixed[end - 1] === '.') end -= 1
  return fixed.slice(0, end)
}

function readGroup(other: LogOtherData): InlineGroupRatio | null {
  const userGroupRatio = other.user_group_ratio
  if (
    userGroupRatio != null &&
    Number.isFinite(userGroupRatio) &&
    userGroupRatio !== -1
  ) {
    return {
      labelKey: 'User Exclusive Ratio',
      value: formatExactNumber(userGroupRatio),
    }
  }
  if (other.group_ratio != null && Number.isFinite(other.group_ratio)) {
    return {
      labelKey: 'Group Ratio',
      value: formatExactNumber(other.group_ratio),
    }
  }
  return null
}

function cacheWriteTerms(
  other: LogOtherData,
  inputPriceUSD: number
): FormulaTerm[] {
  const terms: FormulaTerm[] = []
  const fiveMinute = other.cache_creation_tokens_5m || 0
  const oneHour = other.cache_creation_tokens_1h || 0
  if (
    fiveMinute > 0 &&
    other.cache_creation_ratio_5m != null &&
    Number.isFinite(other.cache_creation_ratio_5m)
  ) {
    terms.push({
      labelKey: 'Cache Write (5m)',
      detailLabelKey: 'Cache Write (5m)',
      tokens: fiveMinute,
      priceUSD: inputPriceUSD * other.cache_creation_ratio_5m,
    })
  }
  if (
    oneHour > 0 &&
    other.cache_creation_ratio_1h != null &&
    Number.isFinite(other.cache_creation_ratio_1h)
  ) {
    terms.push({
      labelKey: 'Cache Write (1h)',
      detailLabelKey: 'Cache Write (1h)',
      tokens: oneHour,
      priceUSD: inputPriceUSD * other.cache_creation_ratio_1h,
    })
  }
  if (terms.length > 0) return terms

  const cacheWrite = other.cache_creation_tokens || 0
  if (
    cacheWrite > 0 &&
    other.cache_creation_ratio != null &&
    Number.isFinite(other.cache_creation_ratio)
  ) {
    terms.push({
      labelKey: 'Cache Write',
      detailLabelKey: 'Cache write price',
      tokens: cacheWrite,
      priceUSD: inputPriceUSD * other.cache_creation_ratio,
    })
  }
  return terms
}

export function buildInlineLogDetails(
  log: InlineLogSource,
  other: LogOtherData | null
): InlineLogDetails {
  const group = other ? readGroup(other) : null
  const ratios: RatioChip[] = []
  let billing: InlineBilling | null = null

  const tiered =
    other?.billing_mode === 'tiered_expr'
      ? getTieredBillingSummary(other)
      : null
  if (tiered && tiered.priceEntries.length > 0) {
    billing = {
      kind: 'tiered',
      prices: tiered.priceEntries.map((entry) => ({
        labelKey: entry.shortLabel,
        priceUSD: entry.price,
      })),
      group,
    }
  } else if (other && isPerCallBilling(other.model_price)) {
    billing = {
      kind: 'per-call',
      priceUSD: other.model_price ?? 0,
      group,
    }
  } else if (other?.model_ratio != null && Number.isFinite(other.model_ratio)) {
    const inputPriceUSD = other.model_ratio * 2
    const promptTokens = log.prompt_tokens || 0
    const completionTokens = log.completion_tokens || 0
    const cacheReadTokens = other.cache_tokens || 0
    const writeTerms = cacheWriteTerms(other, inputPriceUSD)
    const showCacheRead =
      cacheReadTokens > 0 &&
      other.cache_ratio != null &&
      Number.isFinite(other.cache_ratio)
    let inputTokens = promptTokens
    if (!other.claude) {
      const cached =
        (showCacheRead ? cacheReadTokens : 0) +
        writeTerms.reduce((sum, term) => sum + term.tokens, 0)
      const net = promptTokens - cached
      inputTokens = net > 0 ? net : 0
    }

    const terms: FormulaTerm[] = [
      {
        labelKey: 'Input',
        detailLabelKey: 'Input price',
        tokens: inputTokens,
        priceUSD: inputPriceUSD,
      },
    ]
    if (showCacheRead) {
      terms.push({
        labelKey: 'Cache',
        detailLabelKey: 'Cache read price',
        tokens: cacheReadTokens,
        priceUSD: inputPriceUSD * (other.cache_ratio ?? 0),
      })
    }
    terms.push(...writeTerms)
    if (
      other.completion_ratio != null &&
      Number.isFinite(other.completion_ratio)
    ) {
      terms.push({
        labelKey: 'Output',
        detailLabelKey: 'Output price',
        tokens: completionTokens,
        priceUSD: inputPriceUSD * other.completion_ratio,
      })
    }
    billing = {
      kind: 'tokens',
      terms,
      group,
    }
  }

  if (group && billing?.kind !== 'tokens') {
    ratios.push({ labelKey: group.labelKey, value: group.value })
  }

  return {
    ratios,
    billing,
    requestId: log.request_id || '',
    upstreamRequestId: log.upstream_request_id || '',
    requestPath: other?.request_path || '',
    content: log.content || '',
  }
}
