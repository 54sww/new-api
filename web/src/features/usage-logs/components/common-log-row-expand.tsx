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
/* eslint-disable react-refresh/only-export-components */
import type { Row } from '@tanstack/react-table'
import { ChevronRight } from 'lucide-react'
import { createContext, useContext, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'

import { StatusBadge, type StatusBadgeProps } from '@/components/status-badge'
import { formatTimestampToDate } from '@/lib/format'
import { cn } from '@/lib/utils'

import type { UsageLog } from '../data/schema'
import { getLogTypeConfig } from '../lib/utils'

export type CommonLogRowExpandValue = {
  expandedId: string | null
  toggle: (id: string) => void
}

const CommonLogRowExpandContext = createContext<CommonLogRowExpandValue | null>(
  null
)

export function CommonLogRowExpandProvider(props: {
  value: CommonLogRowExpandValue
  children: ReactNode
}) {
  return (
    <CommonLogRowExpandContext.Provider value={props.value}>
      {props.children}
    </CommonLogRowExpandContext.Provider>
  )
}

export function useCommonLogRowExpand(): CommonLogRowExpandValue | null {
  return useContext(CommonLogRowExpandContext)
}

export function commonLogExpandKey(id: number): string {
  return String(id)
}

export function shouldIgnoreRowToggle(target: EventTarget | null): boolean {
  let element: Element | null = null
  if (target instanceof Element) {
    element = target
  } else if (target instanceof Node) {
    element = target.parentElement
  }
  if (!element) return true
  return (
    element.closest(
      'button, a, input, textarea, select, [role="button"], [role="menuitem"], [data-slot="status-badge"]'
    ) != null
  )
}

export function CommonLogTimeCell(props: { row: Row<UsageLog> }) {
  const { t } = useTranslation()
  const expand = useCommonLogRowExpand()
  const log = props.row.original
  const timestamp = props.row.getValue('created_at') as number
  const config = getLogTypeConfig(log.type)
  const expandKey = commonLogExpandKey(log.id)
  const expanded = expand?.expandedId === expandKey

  return (
    <div className='flex min-w-0 items-start gap-1'>
      {expand && (
        <button
          type='button'
          className='text-muted-foreground hover:text-foreground mt-0.5 inline-flex size-4 shrink-0 items-center justify-center rounded-sm'
          aria-expanded={expanded}
          aria-label={expanded ? t('Collapse') : t('Expand')}
          title={expanded ? t('Collapse') : t('Expand')}
          onClick={(event) => {
            event.stopPropagation()
            expand.toggle(expandKey)
          }}
        >
          <ChevronRight
            className={cn(
              'size-3.5 transition-transform',
              expanded && 'rotate-90'
            )}
            aria-hidden='true'
          />
        </button>
      )}
      <div className='flex min-w-0 flex-col gap-0.5'>
        <span className='truncate font-mono text-xs tabular-nums'>
          {formatTimestampToDate(timestamp)}
        </span>
        <StatusBadge
          label={t(config.label)}
          variant={config.color as StatusBadgeProps['variant']}
          size='sm'
          copyable={false}
          className='-ml-1.5 !text-xs [&_span]:!text-xs'
        />
      </div>
    </div>
  )
}
