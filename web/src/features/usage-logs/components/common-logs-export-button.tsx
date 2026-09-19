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
import { getRouteApi } from '@tanstack/react-router'
import axios from 'axios'
import { Download, Loader2 } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import dayjs from '@/lib/dayjs'

import { exportCommonLogs } from '../api'
import { isLogExportOutsideRecentWindow } from '../lib/export-window'
import { buildApiParams } from '../lib/utils'
import { useLogsViewScope } from './usage-logs-provider'

const route = getRouteApi('/_authenticated/usage-logs/$section')

export function CommonLogsExportButton() {
  const { t } = useTranslation()
  const searchParams = route.useSearch()
  const { isAdminView, canManageScope } = useLogsViewScope()
  const [exporting, setExporting] = useState(false)

  const tooltip = canManageScope
    ? t('Export logs matching the current filters')
    : t('Only logs from the last 30 days can be exported')

  const handleExport = async () => {
    const params = buildApiParams({
      page: 1,
      pageSize: 1,
      searchParams,
      isAdmin: isAdminView,
    })
    if (
      isLogExportOutsideRecentWindow({
        isAdmin: canManageScope,
        endTimestamp: params.end_timestamp,
        nowMs: Date.now(),
      })
    ) {
      toast.error(t('Only logs from the last 30 days can be exported'))
      return
    }

    setExporting(true)
    try {
      const response = await exportCommonLogs(params, isAdminView)
      const contentType = String(response.headers['content-type'] ?? '')
      if (contentType.includes('application/json')) {
        const text = await response.data.text()
        let message = ''
        try {
          const body = JSON.parse(text) as { message?: string }
          message = body.message ?? ''
        } catch {
          message = ''
        }
        toast.error(message || t('Export failed'))
        return
      }

      const url = URL.createObjectURL(response.data)
      const link = document.createElement('a')
      link.href = url
      link.download = `usage-logs-${dayjs().format('YYYYMMDD-HHmmss')}.csv`
      link.click()
      window.setTimeout(() => URL.revokeObjectURL(url), 1000)
      if (response.headers['x-log-export-clamped'] === '1') {
        toast.message(t('Exported logs were limited to the last 30 days'))
      }
    } catch (error) {
      if (axios.isAxiosError(error) && error.response?.status === 401) return
      toast.error(t('Export failed'))
    } finally {
      setExporting(false)
    }
  }

  let icon = <Download />
  if (exporting) {
    icon = <Loader2 className='animate-spin' />
  }

  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <Button
            type='button'
            variant='outline'
            onClick={() => {
              void handleExport()
            }}
            disabled={exporting}
            aria-label={t('Export')}
          />
        }
      >
        {icon}
        {t('Export')}
      </TooltipTrigger>
      <TooltipContent>{tooltip}</TooltipContent>
    </Tooltip>
  )
}
