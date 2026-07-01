// WoW Backup Tool v2.0 — Tabbed UI

import {
  DetectInstallations,
  DetectFromPath,
  ValidateInstallation,
  CreateBackup,
  RestoreBackup,
  GetBackupInfo,
  SelectDirectory,
  SelectFile,
  OpenFolder,
  LoadSavedOutputDir,
  SaveOutputDir,
  ShowMessage,
} from '../wailsjs/go/main/App'

import { EventsOn } from '../wailsjs/runtime/runtime'

// ---- Types ----
interface ProgressData { message: string; current: number; total: number }
interface RestoreResult { manifest: unknown; configKept: boolean }
interface WoWInstallation { path: string; version: string; displayName: string }

function $<T extends HTMLElement>(s: string): T { return document.querySelector(s) as T }

// ========================================================
// Shared state
// ========================================================

let installations: WoWInstallation[] = []
let busy = false

// ========================================================
// TAB SWITCHING
// ========================================================

const tabBackup  = $<HTMLButtonElement>('#tab-backup')
const tabRestore = $<HTMLButtonElement>('#tab-restore')
const panelBackup  = $('#panel-backup')
const panelRestore = $('#panel-restore')

tabBackup.addEventListener('click', () => switchTab('backup'))
tabRestore.addEventListener('click', () => switchTab('restore'))

function switchTab(tab: 'backup' | 'restore') {
  const isBackup = tab === 'backup'
  tabBackup.classList.toggle('active', isBackup)
  tabRestore.classList.toggle('active', !isBackup)
  panelBackup.classList.toggle('hidden', !isBackup)
  panelRestore.classList.toggle('hidden', isBackup)
  if (!isBackup) syncRestoreInstallSelect()
}

// ========================================================
// BACKUP PANEL
// ========================================================

const bkInstall = $<HTMLSelectElement>('#bk-install')
const bkStatus   = $<HTMLDivElement>('#bk-status')
const bkRefresh  = $<HTMLButtonElement>('#bk-refresh')
const bkBrowse   = $<HTMLButtonElement>('#bk-browse')
const bkInterface = $<HTMLInputElement>('#bk-interface')
const bkWTF       = $<HTMLInputElement>('#bk-wtf')
const bkFonts     = $<HTMLInputElement>('#bk-fonts')
const bkOutput    = $<HTMLInputElement>('#bk-output')
const bkOutDir    = $<HTMLButtonElement>('#bk-outdir')
const bkProgress  = $<HTMLDivElement>('#bk-progress')
const bkFill      = $<HTMLDivElement>('#bk-fill')
const bkProgText  = $<HTMLDivElement>('#bk-progtext')
const btnBackup    = $<HTMLButtonElement>('#btn-backup')

bkRefresh.addEventListener('click', () => refreshInstallSelect(bkInstall, bkStatus))
bkBrowse.addEventListener('click', () => browseInstallFor(bkInstall, bkStatus))
bkOutDir.addEventListener('click', async () => {
  const p = await SelectDirectory('选择保存目录')
  if (p) { bkOutput.value = p; void SaveOutputDir(p) }
})
bkInstall.addEventListener('change', () => onInstallChanged(bkInstall, bkStatus))
btnBackup.addEventListener('click', doBackup)

async function doBackup() {
  if (busy) return
  const o = bkInstall.options[bkInstall.selectedIndex]
  if (!o || !o.value) { void ShowMessage('warning','提示','请选择 WoW 安装路径'); return }
  const f = getFolders('bk')
  if (!f.length) { void ShowMessage('warning','提示','请至少选择一个文件夹'); return }

  busy = true; setBusy(true)
  bkProgress.classList.remove('hidden')
  bkProgText.textContent = '准备...'; bkFill.style.width = '0%'
  status('创建备份...')

  CreateBackup(o.value, o.dataset.version||'', o.dataset.displayName||'', f, bkOutput.value.trim()||'')
}

// ========================================================
// RESTORE PANEL
// ========================================================

const rsFile      = $<HTMLInputElement>('#rs-file')
const rsChoose    = $<HTMLButtonElement>('#rs-choose')
const rsFileInfo  = $<HTMLDivElement>('#rs-fileinfo')
const rsInstall   = $<HTMLSelectElement>('#rs-install')
const rsStatus    = $<HTMLDivElement>('#rs-status')
const rsRefresh   = $<HTMLButtonElement>('#rs-refresh')
const rsBrowse    = $<HTMLButtonElement>('#rs-browse')
const rsInterface = $<HTMLInputElement>('#rs-interface')
const rsWTF       = $<HTMLInputElement>('#rs-wtf')
const rsFonts     = $<HTMLInputElement>('#rs-fonts')
const rsKeepCfg   = $<HTMLInputElement>('#rs-keepcfg')
const rsProgress  = $<HTMLDivElement>('#rs-progress')
const rsFill      = $<HTMLDivElement>('#rs-fill')
const rsProgText  = $<HTMLDivElement>('#rs-progtext')
const btnRestore  = $<HTMLButtonElement>('#btn-restore')

let selectedBackupPath = ''

rsChoose.addEventListener('click', async () => {
  const p = await SelectFile('选择备份文件')
  if (!p) return
  selectedBackupPath = p
  rsFile.value = p

  // Show backup info
  try {
    const info = await GetBackupInfo(p)
    const folders = info.topFolders as string[] || []
    const date = info.date as string || '未知'
    const size = info.formattedSize as string || '?'
    rsFileInfo.textContent = `日期: ${date}  |  大小: ${size}  |  包含: ${folders.join(', ') || '(空)'}`
    rsFileInfo.className = 'hint ok'
    rsFileInfo.classList.remove('hidden')

    // Auto-check folders present in backup
    rsInterface.checked = folders.includes('Interface')
    rsWTF.checked = folders.includes('WTF')
    rsFonts.checked = folders.includes('Fonts')
  } catch {
    rsFileInfo.textContent = '无法读取备份文件信息'
    rsFileInfo.className = 'hint err'
    rsFileInfo.classList.remove('hidden')
  }

  btnRestore.disabled = false
  btnRestore.textContent = '还原备份'
})

rsRefresh.addEventListener('click', () => refreshInstallSelect(rsInstall, rsStatus))
rsBrowse.addEventListener('click', () => browseInstallFor(rsInstall, rsStatus))
rsInstall.addEventListener('change', () => onInstallChanged(rsInstall, rsStatus))
btnRestore.addEventListener('click', doRestore)

async function doRestore() {
  if (busy || !selectedBackupPath) return
  const o = rsInstall.options[rsInstall.selectedIndex]
  if (!o || !o.value) { void ShowMessage('warning','提示','请选择目标 WoW 安装'); return }

  const f = getFolders('rs')
  if (!f.length) { void ShowMessage('warning','提示','请至少选择一个文件夹'); return }

  busy = true; setBusy(true)
  rsProgress.classList.remove('hidden')
  rsProgText.textContent = '准备...'; rsFill.style.width = '0%'
  status('还原中...')

  RestoreBackup(selectedBackupPath, o.value, o.dataset.version||'', o.dataset.displayName||'',
    rsKeepCfg.checked ? 'true' : 'false', f)
}

function syncRestoreInstallSelect() {
  rsInstall.innerHTML = ''
  for (const i of installations) {
    const o = document.createElement('option')
    o.value = i.path; o.textContent = `${i.displayName} (${i.path})`
    o.dataset.version = i.version; o.dataset.displayName = i.displayName
    rsInstall.appendChild(o)
  }
  if (bkInstall.selectedIndex >= 0) rsInstall.selectedIndex = bkInstall.selectedIndex
  onInstallChanged(rsInstall, rsStatus)
}

// ========================================================
// Shared installation helpers
// ========================================================

async function refreshInstallSelect(sel: HTMLSelectElement, hint: HTMLDivElement) {
  sel.innerHTML = '<option value="">检测中...</option>'
  installations = await DetectInstallations()
  sel.innerHTML = ''
  if (!installations.length) {
    sel.innerHTML = '<option value="">未检测到 — 请手动浏览</option>'
    hint.textContent = '未检测到 WoW 安装'
    hint.className = 'hint err'
    return
  }
  for (const i of installations) {
    const o = document.createElement('option')
    o.value = i.path; o.textContent = `${i.displayName} (${i.path})`
    o.dataset.version = i.version; o.dataset.displayName = i.displayName
    sel.appendChild(o)
  }
  status(`检测到 ${installations.length} 个 WoW 安装`)
  onInstallChanged(sel, hint)
}

function onInstallChanged(sel: HTMLSelectElement, hint: HTMLDivElement) {
  const o = sel.options[sel.selectedIndex]
  if (!o || !o.value) { hint.className = 'hint'; return }
  void ValidateInstallation(o.value).then(v => {
    const p = [(v.interface?'✔':'✘')+' Interface', (v.wtf?'✔':'✘')+' WTF']
    hint.textContent = (v.valid ? '' : '不完整 — ') + p.join('  ')
    hint.className = v.valid ? 'hint ok' : 'hint warn'
  })
}

async function browseInstallFor(sel: HTMLSelectElement, hint: HTMLDivElement) {
  const p = await SelectDirectory('选择 WoW 安装目录')
  if (!p) return
  const inst = await DetectFromPath(p)
  if (!inst) { void ShowMessage('warning','无效路径','未找到 WoW 版本文件夹'); return }
  const label = `${inst.displayName} (${inst.path})`
  for (let i=0; i<sel.options.length; i++) {
    if (sel.options[i].textContent === label) { sel.selectedIndex = i; onInstallChanged(sel, hint); return }
  }
  const o = document.createElement('option')
  o.value = inst.path; o.textContent = label
  o.dataset.version = inst.version; o.dataset.displayName = inst.displayName
  sel.appendChild(o)
  sel.selectedIndex = sel.options.length - 1
  onInstallChanged(sel, hint)
  installations.push(inst as WoWInstallation)
}

// ========================================================
// Helpers
// ========================================================

function getFolders(prefix: 'bk' | 'rs'): string[] {
  const f: string[] = []
  if (prefix === 'bk') {
    if (bkInterface.checked) f.push('Interface')
    if (bkWTF.checked) f.push('WTF')
    if (bkFonts.checked) f.push('Fonts')
  } else {
    if (rsInterface.checked) f.push('Interface')
    if (rsWTF.checked) f.push('WTF')
    if (rsFonts.checked) f.push('Fonts')
  }
  return f
}

function setBusy(b: boolean) {
  const els = [
    bkInstall, bkRefresh, bkBrowse, bkInterface, bkWTF, bkFonts, bkOutput, bkOutDir, btnBackup,
    rsChoose, rsInstall, rsRefresh, rsBrowse, rsInterface, rsWTF, rsFonts, rsKeepCfg, btnRestore,
    tabBackup, tabRestore,
  ]
  for (const e of els) e.disabled = b
  statusBar.classList.toggle('busy', b)
}

const statusBar = $('#status-bar')

function status(msg: string) { statusBar.textContent = msg }

// ========================================================
// Progress events
// ========================================================

function fillBar(bar: HTMLDivElement, d: ProgressData) {
  if (d.total > 0) bar.style.width = (100 * d.current / d.total) + '%'
}

EventsOn('backup:progress', (d: ProgressData) => {
  bkProgText.textContent = d.message; fillBar(bkFill, d)
})
EventsOn('backup:finished', async (path: string) => {
  busy = false; setBusy(false)
  bkProgress.classList.add('hidden'); bkFill.style.width = '0%'
  status('备份完成')
  // Remember the directory for next time
  if (bkOutput.value) void SaveOutputDir(bkOutput.value)
  const r = await ShowMessage('question','备份完成',`备份已创建:\n${path}`)
  if (r === 'yes') void OpenFolder(path)
})
EventsOn('backup:error', async (err: string) => {
  busy = false; setBusy(false)
  bkProgress.classList.add('hidden'); bkFill.style.width = '0%'
  status('备份失败')
  await ShowMessage('error','错误',`备份失败:\n${err}`)
})

EventsOn('restore:progress', (d: ProgressData) => {
  rsProgText.textContent = d.message; fillBar(rsFill, d)
})
EventsOn('restore:finished', async (result: RestoreResult) => {
  busy = false; setBusy(false)
  rsProgress.classList.add('hidden'); rsFill.style.width = '0%'
  status('还原完成')
  if (result && result.configKept)
    await ShowMessage('info','还原完成','备份已成功还原！\n\n已保留当前系统的显示配置。')
  else
    await ShowMessage('info','还原完成','备份已成功还原！')
})
EventsOn('restore:error', async (err: string) => {
  busy = false; setBusy(false)
  rsProgress.classList.add('hidden'); rsFill.style.width = '0%'
  status('还原失败')
  await ShowMessage('error','错误',`还原失败:\n${err}`)
})

// ========================================================
// Boot
// ========================================================

async function init() {
  const saved = await LoadSavedOutputDir()
  if (saved) {
    bkOutput.value = saved
    bkOutput.placeholder = saved
  }
  await refreshInstallSelect(bkInstall, bkStatus)
}

init().catch(err => { console.error(err); status('初始化失败: '+String(err)) })
