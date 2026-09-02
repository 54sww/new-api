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
import { AlertTriangle, ChevronDown, Plus, Search, Trash2 } from 'lucide-react'
import {
  useCallback,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from 'react'
import { useTranslation } from 'react-i18next'

import { searchUsers } from '@/features/users/api'
import type { User } from '@/features/users/types'

import { StaticDataTable } from '@/components/data-table/static/static-data-table'
import { StaticRowActions } from '@/components/data-table/static/static-row-actions'
import { Dialog } from '@/components/dialog'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from '@/components/ui/collapsible'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'

import { safeJsonParse } from '../utils/json-parser'
import { isValidModelPattern } from './utils'

function formatRuleOwner(kind: 'user' | 'group', owner: string): string {
  if (kind === 'user') {
    return `#${owner}`
  }
  return owner
}

type DedicatedModelRatioEditorProps = {
  userModelRatio: string
  groupModelRatio: string
  groupOptions: string[]
  onChange: (field: 'UserModelRatio' | 'GroupModelRatio', value: string) => void
}

function parseNestedRatioMap(value: string): Record<string, Record<string, number>> {
  return safeJsonParse<Record<string, Record<string, number>>>(value, {
    fallback: {},
    silent: true,
  })
}

export function DedicatedModelRatioEditor({
  userModelRatio,
  groupModelRatio,
  groupOptions,
  onChange,
}: DedicatedModelRatioEditorProps) {
  const { t } = useTranslation()

  return (
    <Card className='relative shadow-sm ring-0 before:pointer-events-none before:absolute before:inset-0 before:rounded-xl before:border before:border-border/90'>
      <CardHeader className='border-b bg-muted/20'>
        <CardTitle>{t('Dedicated model ratios')}</CardTitle>
        <CardDescription>
          {t(
            'Configure dedicated billing ratios per user or user group combined with a model pattern. A matched rule replaces the group ratio instead of multiplying it. Priority: user rule > user group rule > special ratio rule > billing group ratio.'
          )}
        </CardDescription>
      </CardHeader>
      <CardContent className='space-y-6'>
        <DedicatedOwnerRules
          kind='user'
          title={t('User rules')}
          description={t(
            'Rules keyed by user ID. Use the exact model name, a "prefix-*" pattern, or "*" as fallback.'
          )}
          value={userModelRatio}
          groupOptions={groupOptions}
          onChange={(value) => onChange('UserModelRatio', value)}
        />
        <DedicatedOwnerRules
          kind='group'
          title={t('User group rules')}
          description={t(
            'Rules keyed by user group. They apply to every user of the group unless a user rule matches first.'
          )}
          value={groupModelRatio}
          groupOptions={groupOptions}
          onChange={(value) => onChange('GroupModelRatio', value)}
        />
      </CardContent>
    </Card>
  )
}

type DedicatedOwnerRulesProps = {
  kind: 'user' | 'group'
  title: string
  description: string
  value: string
  groupOptions: string[]
  onChange: (value: string) => void
}

function DedicatedOwnerRules({
  kind,
  title,
  description,
  value,
  groupOptions,
  onChange,
}: DedicatedOwnerRulesProps) {
  const { t } = useTranslation()
  const [addOwnerOpen, setAddOwnerOpen] = useState(false)
  const [ruleDialogOpen, setRuleDialogOpen] = useState(false)
  const [ruleOwner, setRuleOwner] = useState<string | null>(null)
  const [ruleEditData, setRuleEditData] = useState<{
    pattern: string
    ratio: number
  } | null>(null)

  const owners = useMemo(() => {
    const map = parseNestedRatioMap(value)
    return Object.entries(map)
      .map(([owner, patterns]) => ({
        owner,
        rules: Object.entries(patterns).map(([pattern, ratio]) => ({
          pattern,
          ratio,
        })),
      }))
      .sort((a, b) => a.owner.localeCompare(b.owner, undefined, { numeric: true }))
  }, [value])

  const emitMap = useCallback(
    (map: Record<string, Record<string, number>>) => {
      onChange(JSON.stringify(map, null, 2))
    },
    [onChange]
  )

  const handleOwnerAdd = useCallback(
    (owner: string) => {
      const map = parseNestedRatioMap(value)
      if (!map[owner]) {
        map[owner] = {}
        emitMap(map)
      }
      setAddOwnerOpen(false)
    },
    [value, emitMap]
  )

  const handleOwnerDelete = useCallback(
    (owner: string) => {
      const map = parseNestedRatioMap(value)
      delete map[owner]
      emitMap(map)
    },
    [value, emitMap]
  )

  const handleRuleAdd = useCallback((owner: string) => {
    setRuleOwner(owner)
    setRuleEditData(null)
    setRuleDialogOpen(true)
  }, [])

  const handleRuleEdit = useCallback(
    (owner: string, rule: { pattern: string; ratio: number }) => {
      setRuleOwner(owner)
      setRuleEditData(rule)
      setRuleDialogOpen(true)
    },
    []
  )

  const handleRuleSave = useCallback(
    (pattern: string, ratio: number, oldPattern?: string) => {
      if (!ruleOwner) return
      const map = parseNestedRatioMap(value)
      if (!map[ruleOwner]) {
        map[ruleOwner] = {}
      }
      if (oldPattern && oldPattern !== pattern) {
        delete map[ruleOwner][oldPattern]
      }
      map[ruleOwner][pattern] = ratio
      emitMap(map)
      setRuleDialogOpen(false)
    },
    [ruleOwner, value, emitMap]
  )

  const handleRuleDelete = useCallback(
    (owner: string, pattern: string) => {
      const map = parseNestedRatioMap(value)
      if (map[owner]) {
        delete map[owner][pattern]
        if (Object.keys(map[owner]).length === 0) {
          delete map[owner]
        }
      }
      emitMap(map)
    },
    [value, emitMap]
  )

  const existingOwners = useMemo(
    () => owners.map((entry) => entry.owner),
    [owners]
  )

  const ownerLabel = useCallback(
    (owner: string): ReactNode => {
      if (kind === 'user') {
        return (
          <span className='font-semibold'>
            {t('User #{{id}}', { id: owner })}
          </span>
        )
      }
      return (
        <span className='inline-flex items-center gap-1.5'>
          <span className='font-semibold'>{owner}</span>
          {!groupOptions.includes(owner) && (
            <AlertTriangle
              className='text-destructive h-4 w-4'
              aria-label={t('Not in pricing table')}
            />
          )}
        </span>
      )
    },
    [kind, t, groupOptions]
  )

  return (
    <section className='space-y-3'>
      <div className='flex flex-wrap items-start justify-between gap-2'>
        <div>
          <h3 className='text-sm font-semibold'>{title}</h3>
          <p className='text-muted-foreground text-sm'>{description}</p>
        </div>
        <Button variant='outline' size='sm' onClick={() => setAddOwnerOpen(true)}>
          <Plus className='mr-2 h-4 w-4' />
          {kind === 'user' ? t('Add user') : t('Add user group')}
        </Button>
      </div>

      {owners.length > 0 && (
        <div className='space-y-3'>
          {owners.map((entry) => (
            <Collapsible key={entry.owner}>
              <div className='rounded-lg border'>
                <div className='flex items-center justify-between p-4'>
                  <div className='flex items-center gap-2'>
                    <CollapsibleTrigger render={<Button variant='ghost' size='sm' />}>
                      <ChevronDown className='h-4 w-4' />
                    </CollapsibleTrigger>
                    {ownerLabel(entry.owner)}
                    <span className='text-muted-foreground text-sm'>
                      {t('{{count}} rules', { count: entry.rules.length })}
                    </span>
                  </div>
                  <div className='flex gap-2'>
                    <Button
                      variant='ghost'
                      size='sm'
                      onClick={() => handleRuleAdd(entry.owner)}
                      aria-label={t('Add rule')}
                    >
                      <Plus className='h-4 w-4' />
                    </Button>
                    <Button
                      variant='ghost'
                      size='sm'
                      onClick={() => handleOwnerDelete(entry.owner)}
                      aria-label={t('Delete')}
                    >
                      <Trash2 className='h-4 w-4' />
                    </Button>
                  </div>
                </div>
                <CollapsibleContent>
                  {entry.rules.length > 0 && (
                    <div className='border-t'>
                      <StaticDataTable
                        className='rounded-none border-0'
                        data={entry.rules}
                        getRowKey={(rule) => rule.pattern}
                        columns={[
                          {
                            id: 'pattern',
                            header: t('Model pattern'),
                            cellClassName: 'font-medium',
                            cell: (rule) => (
                              <span className='font-mono text-sm'>{rule.pattern}</span>
                            ),
                          },
                          {
                            id: 'ratio',
                            header: t('Ratio'),
                            cell: (rule) => (
                              <span className='font-medium'>{rule.ratio}</span>
                            ),
                          },
                          {
                            id: 'actions',
                            header: t('Actions'),
                            className: 'text-right',
                            cellClassName: 'text-right',
                            cell: (rule) => (
                              <StaticRowActions
                                editLabel={t('Edit')}
                                deleteLabel={t('Delete')}
                                menuLabel={t('Open menu')}
                                onEdit={() => handleRuleEdit(entry.owner, rule)}
                                onDelete={() => handleRuleDelete(entry.owner, rule.pattern)}
                              />
                            ),
                          },
                        ]}
                      />
                    </div>
                  )}
                </CollapsibleContent>
              </div>
            </Collapsible>
          ))}
        </div>
      )}

      {kind === 'user' ? (
        <UserPickerDialog
          open={addOwnerOpen}
          onOpenChange={setAddOwnerOpen}
          existingOwners={existingOwners}
          onSelect={handleOwnerAdd}
        />
      ) : (
        <GroupPickerDialog
          open={addOwnerOpen}
          onOpenChange={setAddOwnerOpen}
          groupOptions={groupOptions}
          existingOwners={existingOwners}
          onSelect={handleOwnerAdd}
        />
      )}

      <DedicatedRuleDialog
        open={ruleDialogOpen}
        onOpenChange={setRuleDialogOpen}
        onSave={handleRuleSave}
        editData={ruleEditData}
        ownerLabel={
          ruleOwner === null ? null : formatRuleOwner(kind, ruleOwner)
        }
      />
    </section>
  )
}

type UserPickerDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  existingOwners: string[]
  onSelect: (userId: string) => void
}

function UserPickerDialog({
  open,
  onOpenChange,
  existingOwners,
  onSelect,
}: UserPickerDialogProps) {
  const { t } = useTranslation()
  const [keyword, setKeyword] = useState('')
  const [results, setResults] = useState<User[]>([])
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    if (!open) {
      setKeyword('')
      setResults([])
      setLoading(false)
    }
  }, [open])

  const handleSearch = useCallback(async () => {
    setLoading(true)
    try {
      const res = await searchUsers({ keyword: keyword.trim(), page_size: 20 })
      setResults(res.data?.items ?? [])
    } catch {
      setResults([])
    } finally {
      setLoading(false)
    }
  }, [keyword])

  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      title={t('Add user')}
      description={t(
        'Search users by username or display name. Rules are stored by user ID, so renaming a user later keeps the rule.'
      )}
      contentHeight='auto'
      bodyClassName='space-y-4'
      footer={
        <Button variant='outline' onClick={() => onOpenChange(false)}>
          {t('Cancel')}
        </Button>
      }
    >
      <div className='space-y-4 py-4'>
        <div className='flex gap-2'>
          <Input
            value={keyword}
            onChange={(event) => setKeyword(event.target.value)}
            onKeyDown={(event) => {
              if (event.key === 'Enter') {
                event.preventDefault()
                void handleSearch()
              }
            }}
            placeholder={t('Search by username or display name')}
          />
          <Button onClick={() => void handleSearch()} disabled={loading}>
            <Search className='mr-2 h-4 w-4' />
            {loading ? t('Loading...') : t('Search users')}
          </Button>
        </div>
        {results.length > 0 && (
          <div className='max-h-64 space-y-1 overflow-y-auto'>
            {results.map((user) => {
              const ownerKey = String(user.id)
              const added = existingOwners.includes(ownerKey)
              return (
                <button
                  key={user.id}
                  type='button'
                  disabled={added}
                  className='hover:bg-muted flex w-full items-center justify-between rounded-md border px-3 py-2 text-left text-sm disabled:cursor-not-allowed disabled:opacity-60'
                  onClick={() => onSelect(ownerKey)}
                >
                  <span>
                    <span className='font-medium'>{user.username}</span>
                    {user.display_name && user.display_name !== user.username && (
                      <span className='text-muted-foreground ml-2'>
                        {user.display_name}
                      </span>
                    )}
                  </span>
                  <span className='text-muted-foreground text-xs'>
                    {added ? t('Added') : `#${user.id}`}
                  </span>
                </button>
              )
            })}
          </div>
        )}
      </div>
    </Dialog>
  )
}

type GroupPickerDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  groupOptions: string[]
  existingOwners: string[]
  onSelect: (group: string) => void
}

function GroupPickerDialog({
  open,
  onOpenChange,
  groupOptions,
  existingOwners,
  onSelect,
}: GroupPickerDialogProps) {
  const { t } = useTranslation()
  const [group, setGroup] = useState<string | null>(null)

  useEffect(() => {
    if (!open) {
      setGroup(null)
    }
  }, [open])

  const options = useMemo(() => {
    const extras = existingOwners.filter((name) => !groupOptions.includes(name))
    return [...groupOptions, ...extras]
  }, [groupOptions, existingOwners])

  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      title={t('Add user group')}
      description={t(
        'Pick the user group whose members will get dedicated model ratios.'
      )}
      contentHeight='auto'
      bodyClassName='space-y-4'
      footer={
        <>
          <Button variant='outline' onClick={() => onOpenChange(false)}>
            {t('Cancel')}
          </Button>
          <Button
            disabled={!group || existingOwners.includes(group)}
            onClick={() => group && onSelect(group)}
          >
            {t('Add')}
          </Button>
        </>
      }
    >
      <div className='space-y-4 py-4'>
        <div className='space-y-2'>
          <Label>{t('User group name')}</Label>
          <Select
            value={group}
            onValueChange={(value) => {
              if (typeof value === 'string' && value !== '') setGroup(value)
            }}
          >
            <SelectTrigger className='w-full'>
              <SelectValue placeholder={t('Select a group')} />
            </SelectTrigger>
            <SelectContent alignItemWithTrigger={false}>
              <SelectGroup>
                {options.map((name) => (
                  <SelectItem key={name} value={name}>
                    {name}
                  </SelectItem>
                ))}
              </SelectGroup>
            </SelectContent>
          </Select>
        </div>
      </div>
    </Dialog>
  )
}

type DedicatedRuleDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  onSave: (pattern: string, ratio: number, oldPattern?: string) => void
  editData: { pattern: string; ratio: number } | null
  ownerLabel: string | null
}

function DedicatedRuleDialog({
  open,
  onOpenChange,
  onSave,
  editData,
  ownerLabel,
}: DedicatedRuleDialogProps) {
  const { t } = useTranslation()
  const [pattern, setPattern] = useState('')
  const [ratio, setRatio] = useState('')
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!open) {
      setPattern('')
      setRatio('')
      setError(null)
      return
    }
    setPattern(editData?.pattern ?? '')
    setRatio(editData ? String(editData.ratio) : '')
    setError(null)
  }, [editData, open])

  const handleSave = () => {
    const trimmed = pattern.trim()
    if (!isValidModelPattern(trimmed)) {
      setError(t('Invalid model pattern'))
      return
    }
    const parsedRatio = Number.parseFloat(ratio)
    if (Number.isNaN(parsedRatio) || parsedRatio < 0) {
      setError(t('Ratio must be a non-negative number'))
      return
    }
    onSave(trimmed, parsedRatio, editData?.pattern)
  }

  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      title={editData ? t('Edit rule') : t('Add rule')}
      description={t(
        'The model pattern supports the exact model name, a "prefix-*" wildcard, or "*" as the fallback. Exact match wins, then the longest prefix, then "*".'
      )}
      contentHeight='auto'
      bodyClassName='space-y-4'
      footer={
        <>
          <Button variant='outline' onClick={() => onOpenChange(false)}>
            {t('Cancel')}
          </Button>
          <Button onClick={handleSave}>{editData ? t('Update') : t('Add')}</Button>
        </>
      }
    >
      <div className='space-y-4 py-4'>
        {ownerLabel && (
          <p className='text-muted-foreground text-sm'>
            {t('Applies to {{owner}}', { owner: ownerLabel })}
          </p>
        )}
        <div className='space-y-2'>
          <Label>{t('Model pattern')}</Label>
          <Input
            value={pattern}
            onChange={(event) => setPattern(event.target.value)}
            placeholder='deepseek-*'
          />
        </div>
        <div className='space-y-2'>
          <Label>{t('Ratio')}</Label>
          <Input
            value={ratio}
            onChange={(event) => {
              const val = event.target.value
              if (val === '' || !Number.isNaN(Number.parseFloat(val))) {
                setRatio(val)
              }
            }}
            placeholder='0.8'
          />
        </div>
        {error && <p className='text-destructive text-sm'>{error}</p>}
      </div>
    </Dialog>
  )
}
