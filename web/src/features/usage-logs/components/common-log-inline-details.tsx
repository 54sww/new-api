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
import { Check, Copy } from 'lucide-react'
import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { useCopyToClipboard } from '@/hooks/use-copy-to-clipboard'
import { formatBillingCurrencyFromUSD } from '@/lib/currency'
import { formatLogQuota } from '@/lib/format'

import type { UsageLog } from '../data/schema'
import { parseLogOther } from '../lib/format'
import {
  buildInlineLogDetails,
  type InlineBilling,
  type RatioChip,
  type TokenBilling,
} from '../lib/inline-log-details'

const priceFormat = {
  digitsLarge: 6,
  digitsSmall: 6,
  abbreviate: false,
} as const

function formatPrice(usd: number): string {
  return formatBillingCurrencyFromUSD(usd, priceFormat)
}

function CopyableValue(props: { label: string; value: string }) {
  const { t } = useTranslation()
  const { copiedText, copyToClipboard } = useCopyToClipboard({ notify: false })
  const copied = copiedText === props.value

  return (
    <div className='flex min-w-0 items-start gap-2'>
      <span className='text-muted-foreground w-28 shrink-0'>{props.label}</span>
      <span className='min-w-0 font-mono break-all'>{props.value}</span>
      <Button
        type='button'
        variant='ghost'
        size='sm'
        className='size-5 shrink-0 p-0'
        onClick={() => copyToClipboard(props.value)}
        title={t('Copy to clipboard')}
        aria-label={t('Copy to clipboard')}
      >
        {copied ? (
          <Check className='size-3 text-green-600' />
        ) : (
          <Copy className='size-3' />
        )}
      </Button>
    </div>
  )
}

function RatioChips(props: { ratios: RatioChip[] }) {
  const { t } = useTranslation()
  if (props.ratios.length === 0) return null

  return (
    <section className='space-y-1.5'>
      <p className='font-semibold'>{t('Log Details')}</p>
      <div className='flex flex-wrap gap-1.5'>
        {props.ratios.map((item) => (
          <span
            key={`${item.labelKey}-${item.value}`}
            className='bg-muted inline-flex items-center gap-1 rounded-md px-2 py-0.5'
          >
            <span className='text-muted-foreground'>{t(item.labelKey)}</span>
            <span className='font-mono'>{item.value}</span>
          </span>
        ))}
      </div>
    </section>
  )
}

function TokenBillingRows(props: { billing: TokenBilling; cost: string }) {
  const { t } = useTranslation()
  const separator = t(', ')
  const detailLabelOrder = [
    'Input price',
    'Output price',
    'Cache read price',
    'Cache write price',
    'Cache Write',
    'Cache Write (5m)',
    'Cache Write (1h)',
  ]
  const detailTerms = detailLabelOrder.flatMap((labelKey) =>
    props.billing.terms.filter((term) => term.detailLabelKey === labelKey)
  )
  const detailParts = detailTerms.map((term) =>
    t('{{label}} {{price}} / 1M tokens', {
      label: t(term.detailLabelKey),
      price: formatPrice(term.priceUSD),
    })
  )
  if (props.billing.group) {
    detailParts.push(
      `${t(props.billing.group.labelKey)} ${props.billing.group.value}x`
    )
  }
  const expression = props.billing.terms
    .map((term) =>
      t('{{label}} {{tokens}} tokens / 1M tokens * {{price}}', {
        label: t(term.labelKey),
        tokens: String(term.tokens),
        price: formatPrice(term.priceUSD),
      })
    )
    .join(' + ')
  let formula = `(${expression})`
  if (props.billing.group) {
    formula += ` * ${t(props.billing.group.labelKey)} ${props.billing.group.value}`
  }
  formula += ` = ${props.cost}`

  return (
    <div className='grid grid-cols-[auto_minmax(0,1fr)] gap-x-3 gap-y-1.5'>
      <p className='font-semibold whitespace-nowrap'>{t('Log Details')}</p>
      <p className='break-all'>{detailParts.join(separator)}</p>
      <p className='font-semibold whitespace-nowrap'>{t('Billing Process')}</p>
      <p className='font-mono break-all'>{formula}</p>
    </div>
  )
}

function BillingProcess(props: { log: UsageLog; billing: InlineBilling }) {
  const { t } = useTranslation()
  const cost = formatLogQuota(props.log.quota)
  if (props.billing.kind === 'tokens') {
    return <TokenBillingRows billing={props.billing} cost={cost} />
  }

  let body: ReactNode = null
  if (props.billing.kind === 'per-call') {
    body = (
      <div className='space-y-1'>
        <p>
          {t('Per-call')} · {formatPrice(props.billing.priceUSD)}
        </p>
        <p>
          {t('Total Cost')}: {cost}
        </p>
      </div>
    )
  } else {
    body = (
      <div className='space-y-1'>
        <p>{t('Dynamic Pricing')}</p>
        {props.billing.prices.map((entry) => (
          <p key={entry.labelKey}>
            {t(entry.labelKey)} {formatPrice(entry.priceUSD)}/M
          </p>
        ))}
        <p>
          {t('Total Cost')}: {cost}
        </p>
      </div>
    )
  }

  return (
    <section className='space-y-1.5'>
      <p className='font-semibold'>{t('Billing Process')}</p>
      {body}
      <p className='text-muted-foreground'>
        {t('For reference only. Actual charges prevail.')}
      </p>
    </section>
  )
}

export function CommonLogInlineDetails(props: {
  log: UsageLog
  isAdmin: boolean
}) {
  const { t } = useTranslation()
  const details = buildInlineLogDetails(
    props.log,
    parseLogOther(props.log.other)
  )
  const showPath = props.isAdmin && details.requestPath !== ''
  const showContent = details.billing == null && details.content !== ''
  const hasAnything =
    details.ratios.length > 0 ||
    details.billing != null ||
    details.requestId !== '' ||
    details.upstreamRequestId !== '' ||
    showPath ||
    showContent

  return (
    <div className='space-y-3 text-xs leading-relaxed' data-inline-log-details>
      {details.ratios.length > 0 && <RatioChips ratios={details.ratios} />}
      {details.billing && (
        <BillingProcess log={props.log} billing={details.billing} />
      )}
      {details.requestId !== '' && (
        <CopyableValue label={t('Request ID')} value={details.requestId} />
      )}
      {details.upstreamRequestId !== '' && (
        <CopyableValue
          label={t('Upstream Request ID')}
          value={details.upstreamRequestId}
        />
      )}
      {showPath && (
        <div className='flex min-w-0 items-start gap-2'>
          <span className='text-muted-foreground w-28 shrink-0'>
            {t('Request Path')}
          </span>
          <span className='min-w-0 font-mono break-all'>
            {details.requestPath}
          </span>
        </div>
      )}
      {showContent && (
        <section className='space-y-1'>
          <p className='font-semibold'>{t('Content')}</p>
          <p className='max-h-40 overflow-auto break-all whitespace-pre-wrap'>
            {details.content}
          </p>
        </section>
      )}
      {!hasAnything && (
        <p className='text-muted-foreground'>{t('No additional details')}</p>
      )}
    </div>
  )
}
