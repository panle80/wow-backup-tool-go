// WoW Backup Tool v2.1.0 — Tabbed UI

import {
  DetectInstallations,
  DetectAllFromPath,
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
let currentInstall: WoWInstallation | null = null
let busy = false

// ========================================================
// GLOBAL GAME SELECTOR
// ========================================================

const gameName   = $<HTMLDivElement>('#game-name')
const gamePath   = $<HTMLDivElement>('#game-path')
const gameStatus = $<HTMLDivElement>('#game-status')
const gameSwitch = $<HTMLButtonElement>('#game-switch')

const gameModal      = $<HTMLDivElement>('#game-modal')
const gameModalClose = $<HTMLButtonElement>('#game-modal-close')
const gameList       = $<HTMLDivElement>('#game-list')
const gameRefresh    = $<HTMLButtonElement>('#game-refresh')
const gameBrowse     = $<HTMLButtonElement>('#game-browse')

gameSwitch.addEventListener('click', openGameModal)
gameModalClose.addEventListener('click', closeGameModal)
gameRefresh.addEventListener('click', () => {
  void refreshInstallations().catch(err => status('检测失败: ' + String(err)))
})
gameBrowse.addEventListener('click', () => void browseForInstall())
gameModal.addEventListener('click', e => {
  if (e.target === gameModal) closeGameModal()
})
// Keep keyboard focus inside the picker while it is open.
gameModal.addEventListener('keydown', e => {
  if (e.key !== 'Tab') return
  const focusables = Array.from(
    gameModal.querySelectorAll<HTMLElement>('button:not([disabled])'),
  )
  if (!focusables.length) return
  const first = focusables[0]
  const last = focusables[focusables.length - 1]
  if (e.shiftKey && document.activeElement === first) {
    e.preventDefault()
    last.focus()
  } else if (!e.shiftKey && document.activeElement === last) {
    e.preventDefault()
    first.focus()
  }
})
document.addEventListener('keydown', e => {
  if (e.key === 'Escape') closeGameModal()
})

function samePath(a: string, b: string): boolean {
  return a.toLowerCase() === b.toLowerCase()
}

type Validation = Record<string, boolean>

// Cache ValidateInstallation results so re-rendering the picker does not fire
// one IPC call per installation every time it opens.
const validationCache = new Map<string, Promise<Validation>>()

function validateCached(path: string): Promise<Validation> {
  const key = path.toLowerCase()
  let pending = validationCache.get(key)
  if (!pending) {
    pending = ValidateInstallation(path).catch(err => {
      validationCache.delete(key)
      throw err
    })
    validationCache.set(key, pending)
  }
  return pending
}

function isGameModalOpen(): boolean {
  return !gameModal.classList.contains('hidden')
}

async function refreshInstallations() {
  gameName.textContent = '正在检测...'
  gamePath.textContent = ''
  gamePath.title = ''
  gameStatus.textContent = ''
  gameStatus.className = 'hint'

  let detectError = ''
  try {
    installations = (await DetectInstallations()) || []
  } catch (err) {
    installations = []
    detectError = String(err)
  }

  // Keep the selected game stable across refreshes. Only pick a default when
  // nothing is selected yet; a selection made while the scan was running (or
  // a manually browsed install) must never be silently replaced, otherwise a
  // later restore would target the wrong installation.
  if (!currentInstall && installations.length) {
    currentInstall = installations[0]
  }
  const selected = currentInstall
  if (selected && !installations.some(i => samePath(i.path, selected.path))) {
    installations.push(selected)
  }

  // Paths may have appeared/disappeared since the last scan.
  validationCache.clear()
  updateGameCard()
  if (isGameModalOpen()) renderGameList()
  if (detectError) {
    status('检测失败: ' + detectError)
  } else {
    status(installations.length
      ? `检测到 ${installations.length} 个 WoW 安装`
      : '未检测到 WoW 安装，请手动选择目录')
  }
}

function updateGameCard() {
  gameStatus.textContent = ''
  gameStatus.className = 'hint'

  if (!currentInstall) {
    gameName.textContent = '未检测到 WoW'
    gamePath.textContent = '点击右侧“切换”手动选择安装目录'
    gamePath.title = ''
    return
  }

  gameName.textContent = currentInstall.displayName
  gamePath.textContent = currentInstall.path
  gamePath.title = currentInstall.path

  const path = currentInstall.path
  void validateCached(path).then(v => {
    if (!currentInstall || !samePath(currentInstall.path, path)) return
    const parts = [(v.interface ? '✔' : '✘') + ' Interface', (v.wtf ? '✔' : '✘') + ' WTF']
    gameStatus.textContent = (v.valid ? '' : '不完整 — ') + parts.join('  ')
    gameStatus.className = 'hint ' + (v.valid ? 'ok' : 'warn')
  }).catch(() => {
    if (!currentInstall || !samePath(currentInstall.path, path)) return
    gameStatus.textContent = '无法验证安装状态'
    gameStatus.className = 'hint warn'
  })
}

function renderGameList() {
  gameList.innerHTML = ''
  if (!installations.length) {
    gameList.innerHTML = '<div class="game-empty">未检测到 WoW 安装<br>点击下方“浏览其他目录...”手动选择</div>'
    return
  }

  for (const inst of installations) {
    const item = document.createElement('button')
    item.type = 'button'
    item.className = 'game-item' +
      (currentInstall && samePath(currentInstall.path, inst.path) ? ' current' : '')
    item.innerHTML =
      '<span class="game-item-dot"></span>' +
      '<span class="game-item-main">' +
        '<span class="game-item-name"></span>' +
        '<span class="game-item-path"></span>' +
      '</span>' +
      '<span class="game-item-badge"></span>'

    ;(item.querySelector('.game-item-name') as HTMLElement).textContent = inst.displayName
    ;(item.querySelector('.game-item-path') as HTMLElement).textContent = inst.path
    const badge = item.querySelector('.game-item-badge') as HTMLElement
    item.addEventListener('click', () => selectInstall(inst))
    gameList.appendChild(item)

    void validateCached(inst.path).then(v => {
      if (!badge.isConnected) return
      badge.textContent = v.valid ? '可用' : '不完整'
      badge.classList.toggle('incomplete', !v.valid)
    }).catch(() => {
      if (!badge.isConnected) return
      badge.textContent = '未知'
      badge.classList.add('incomplete')
    })
  }
}

let lastFocused: HTMLElement | null = null

function selectInstall(inst: WoWInstallation) {
  currentInstall = inst
  updateGameCard()
  closeGameModal()
  status(`当前游戏：${inst.displayName}`)
}

function openGameModal() {
  if (busy) return
  lastFocused = document.activeElement as HTMLElement | null
  renderGameList()
  gameModal.classList.remove('hidden')
  const first = gameModal.querySelector<HTMLElement>('.game-item, #game-modal-close')
  first?.focus()
}

function closeGameModal() {
  if (!isGameModalOpen()) return
  gameModal.classList.add('hidden')
  lastFocused?.focus()
  lastFocused = null
}

async function browseForInstall() {
  try {
    const p = await SelectDirectory('选择 WoW 安装目录')
    if (!p) return

    const found = (await DetectAllFromPath(p)) || []
    if (!found.length) {
      void ShowMessage('warning', '无效路径', '未找到 WoW 版本文件夹')
      return
    }

    // A WoW root folder can contain several version folders (e.g. _retail_,
    // _classic_ and _forever_), so add every version to the picker.
    for (const inst of found) {
      if (!installations.some(i => samePath(i.path, inst.path))) {
        installations.push(inst)
      }
    }

    if (found.length === 1) {
      selectInstall(found[0])
    } else {
      renderGameList()
      status(`该目录下检测到 ${found.length} 个版本，请选择`)
    }
  } catch (err) {
    status('浏览失败: ' + String(err))
    void ShowMessage('error', '错误', '无法读取该目录:\n' + String(err))
  }
}

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
}

// ========================================================
// BACKUP PANEL
// ========================================================

const bkInterface = $<HTMLInputElement>('#bk-interface')
const bkWTF       = $<HTMLInputElement>('#bk-wtf')
const bkFonts     = $<HTMLInputElement>('#bk-fonts')
const bkOutput    = $<HTMLInputElement>('#bk-output')
const bkOutDir    = $<HTMLButtonElement>('#bk-outdir')
const bkProgress  = $<HTMLDivElement>('#bk-progress')
const bkFill      = $<HTMLDivElement>('#bk-fill')
const bkProgText  = $<HTMLDivElement>('#bk-progtext')
const btnBackup   = $<HTMLButtonElement>('#btn-backup')

bkOutDir.addEventListener('click', async () => {
  const p = await SelectDirectory('选择保存目录')
  if (p) { bkOutput.value = p; void SaveOutputDir(p) }
})
btnBackup.addEventListener('click', doBackup)

function doBackup() {
  if (busy) return
  if (!currentInstall) { void ShowMessage('warning','提示','请先选择 WoW 安装'); return }
  const f = getFolders('bk')
  if (!f.length) { void ShowMessage('warning','提示','请至少选择一个文件夹'); return }

  closeGameModal()
  busy = true; setBusy(true)
  bkProgress.classList.remove('hidden')
  bkProgText.textContent = '准备...'; bkFill.style.width = '0%'
  status('创建备份...')

  CreateBackup(
    currentInstall.path,
    currentInstall.version,
    currentInstall.displayName,
    f,
    bkOutput.value.trim() || '',
  )
}

// ========================================================
// RESTORE PANEL
// ========================================================

const rsFile      = $<HTMLInputElement>('#rs-file')
const rsChoose    = $<HTMLButtonElement>('#rs-choose')
const rsFileInfo  = $<HTMLDivElement>('#rs-fileinfo')
const rsInterface = $<HTMLInputElement>('#rs-interface')
const rsWTF       = $<HTMLInputElement>('#rs-wtf')
const rsFonts     = $<HTMLInputElement>('#rs-fonts')
const rsKeepCfg   = $<HTMLInputElement>('#rs-keepcfg')
const rsProgress  = $<HTMLDivElement>('#rs-progress')
const rsFill      = $<HTMLDivElement>('#rs-fill')
const rsProgText  = $<HTMLDivElement>('#rs-progtext')
const btnRestore  = $<HTMLButtonElement>('#btn-restore')

let selectedBackupPath = ''
let backupReady = false

rsChoose.addEventListener('click', async () => {
  const p = await SelectFile('选择备份文件')
  if (!p) return
  selectedBackupPath = p
  rsFile.value = p
  backupReady = false
  btnRestore.disabled = true
  btnRestore.textContent = '正在读取备份...'

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

    backupReady = true
    btnRestore.disabled = false
    btnRestore.textContent = '还原备份'
  } catch {
    // Never enable restore for an unreadable archive: the backend would
    // delete the target folders before discovering the file is invalid.
    rsFileInfo.textContent = '无法读取备份文件信息'
    rsFileInfo.className = 'hint err'
    rsFileInfo.classList.remove('hidden')
    btnRestore.disabled = true
    btnRestore.textContent = '备份文件无效，请重新选择'
  }
})

btnRestore.addEventListener('click', doRestore)

function doRestore() {
  if (busy || !selectedBackupPath || !backupReady) return
  if (!currentInstall) { void ShowMessage('warning','提示','请先选择 WoW 安装'); return }

  const f = getFolders('rs')
  if (!f.length) { void ShowMessage('warning','提示','请至少选择一个文件夹'); return }

  closeGameModal()
  busy = true; setBusy(true)
  rsProgress.classList.remove('hidden')
  rsProgText.textContent = '准备...'; rsFill.style.width = '0%'
  status('还原中...')

  RestoreBackup(
    selectedBackupPath,
    currentInstall.path,
    currentInstall.version,
    currentInstall.displayName,
    rsKeepCfg.checked ? 'true' : 'false',
    f,
  )
}

// ========================================================
// Helpers
// ========================================================

function showDone(title: string, detail = '', path = ''): Promise<void> {
  return new Promise(resolve => {
    const overlay = document.createElement('div')
    overlay.className = 'done-overlay'

    const box = document.createElement('div')
    box.className = 'done-box'

    const heading = document.createElement('p')
    heading.className = 'done-title'
    heading.textContent = title
    box.appendChild(heading)

    if (detail) {
      const d = document.createElement('p')
      d.className = 'done-detail'
      d.textContent = detail
      box.appendChild(d)
    }

    if (path) {
      const p = document.createElement('div')
      p.className = 'done-path'
      p.textContent = path
      p.title = path
      box.appendChild(p)
    }

    const btn = document.createElement('button')
    btn.className = 'btn-primary'
    btn.textContent = '完成'
    btn.addEventListener('click', () => {
      overlay.remove(); resolve()
    })
    box.appendChild(btn)

    overlay.appendChild(box)
    document.body.appendChild(overlay)
  })
}

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
    gameSwitch,
    bkInterface, bkWTF, bkFonts, bkOutput, bkOutDir, btnBackup,
    rsChoose, rsInterface, rsWTF, rsFonts, rsKeepCfg, btnRestore,
    tabBackup, tabRestore,
  ]
  for (const e of els) e.disabled = b
  // Keep restore disabled unless a readable backup is selected, even after a
  // backup/restore finishes and setBusy(false) re-enables the panel.
  if (!b) btnRestore.disabled = !backupReady
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
  await showDone('备份已创建', '', path)
  void OpenFolder(path)
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
    await showDone('备份已成功还原', '已保留当前系统的显示配置')
  else
    await showDone('备份已成功还原')
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
  await refreshInstallations()
}

init().catch(err => { console.error(err); status('初始化失败: '+String(err)) })
