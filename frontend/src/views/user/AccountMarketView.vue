<template>
  <AppLayout>
    <div class="market-page">
      <header class="page-heading">
        <div>
          <p class="eyebrow">RESOURCE CONTRIBUTION</p>
          <h1>我的账号</h1>
          <p>上传账号参与平台调度，调用产生的收益可申请提现。</p>
        </div>
        <button class="btn btn-primary" :disabled="!config?.enabled" @click="openUpload">
          <Icon name="upload" size="sm" />
          上传账号
        </button>
      </header>

      <div v-if="config && !config.enabled" class="notice warning">
        <Icon name="exclamationTriangle" size="sm" />
        当前暂未开放账号上传，已有账号和收益仍可查看。
      </div>

      <section class="summary-strip">
        <div class="summary-primary">
          <span>可提现收益</span>
          <strong>{{ money(summary.available) }}</strong>
          <button class="btn btn-primary btn-sm" :disabled="summary.available <= 0" @click="openWithdrawal">
            申请提现
          </button>
        </div>
        <div><span>累计收益</span><strong>{{ money(summary.total) }}</strong></div>
        <div><span>提现处理中</span><strong>{{ money(summary.locked) }}</strong></div>
        <div><span>已提现</span><strong>{{ money(summary.withdrawn) }}</strong></div>
        <div><span>贡献账号</span><strong>{{ summary.account_count }}</strong></div>
      </section>

      <div class="view-tabs" role="tablist">
        <button :class="{ active: activeTab === 'accounts' }" @click="activeTab = 'accounts'">账号</button>
        <button :class="{ active: activeTab === 'withdrawals' }" @click="activeTab = 'withdrawals'">提现记录</button>
      </div>

      <section v-if="activeTab === 'accounts'" class="panel">
        <div class="panel-head">
          <div><h2>已上传账号</h2><p>状态与额度来自平台当前账号信息。</p></div>
          <button class="icon-action" title="刷新" :disabled="loading" @click="loadAll"><Icon name="refresh" :class="{ 'animate-spin': loading }" /></button>
        </div>

        <div v-if="loading" class="empty-state">正在读取账号...</div>
        <div v-else-if="accounts.length === 0" class="empty-state">
          <Icon name="inbox" size="xl" />
          <strong>还没有贡献账号</strong>
          <span>上传 Sub2API JSON 后，账号会进入管理员指定分组。</span>
        </div>
        <div v-else class="account-list">
          <article v-for="account in accounts" :key="account.account_id" class="account-row">
            <div class="account-identity">
              <span class="platform-mark">{{ account.platform.slice(0, 2).toUpperCase() }}</span>
              <div>
                <div class="identity-line">
                  <strong>{{ account.name }}</strong>
                  <span class="status-dot" :class="statusClass(account)">{{ statusText(account) }}</span>
                </div>
                <p>
                  {{ account.email || '未识别邮箱' }} · {{ account.type }}
                  <template v-if="account.proxy_name"> · {{ account.proxy_name }}（{{ account.proxy_protocol?.toUpperCase() }}）</template>
                </p>
              </div>
            </div>
            <div class="account-metrics">
              <div><span>剩余额度</span><strong>{{ displayValue(account.quota) }}</strong></div>
              <div><span>重置时间</span><strong>{{ displayDate(account.reset_at) }}</strong></div>
              <div><span>并发容量</span><strong>{{ account.concurrency }}</strong></div>
              <div><span>所属分组</span><strong>{{ account.group_name || '-' }}</strong></div>
              <div><span>分组倍率</span><strong>{{ number(account.group_rate_multiplier) }}x</strong></div>
              <div><span>累计收益</span><strong class="earning">{{ money(account.total_earned) }}</strong></div>
              <AccountMarketQuotaCell v-if="account.platform === 'openai' && account.type === 'oauth'" :account-id="account.account_id" />
            </div>
            <p v-if="account.error_message" class="account-error">{{ account.error_message }}</p>
            <div class="account-actions">
              <label class="switch-control">
                <input type="checkbox" :checked="account.schedulable" :disabled="busyId === account.account_id || !['active', 'paused'].includes(account.market_status)" @change="toggleSchedule(account)" />
                <span></span>{{ account.schedulable ? '参与调度' : '暂停调度' }}
              </label>
              <button class="btn btn-secondary btn-sm" @click="openEdit(account)"><Icon name="edit" size="sm" />编辑</button>
              <button class="btn btn-secondary btn-sm" @click="downloadAccount(account)"><Icon name="download" size="sm" />导出</button>
              <button class="btn btn-danger btn-sm" @click="remove(account)"><Icon name="trash" size="sm" />移除</button>
            </div>
          </article>
        </div>
      </section>

      <section v-else class="panel">
        <div class="panel-head"><div><h2>提现记录</h2><p>抽佣比例在提交申请时固定，不受后续配置修改影响。</p></div></div>
        <div v-if="withdrawals.length === 0" class="empty-state">暂无提现记录</div>
        <div v-else class="table-wrap">
          <table>
            <thead><tr><th>提交时间</th><th>申请金额</th><th>平台抽佣</th><th>实际到账</th><th>收款信息</th><th>状态</th><th></th></tr></thead>
            <tbody>
              <tr v-for="item in withdrawals" :key="item.id">
                <td>{{ dateTime(item.created_at) }}</td>
                <td>{{ money(item.gross_amount) }}</td>
                <td>{{ money(item.commission_amount) }}（{{ number(item.commission_percent) }}%）</td>
                <td class="earning">{{ money(item.net_amount) }}</td>
                <td><span class="payout">{{ item.payout_method }} · {{ item.payout_account }}</span></td>
                <td><span class="withdraw-status" :class="item.status">{{ withdrawalStatus(item.status) }}</span><small v-if="item.review_note">{{ item.review_note }}</small></td>
                <td><button v-if="item.status === 'pending'" class="btn btn-secondary btn-sm" @click="cancel(item.id)">取消</button></td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>
    </div>

    <BaseDialog :show="uploadOpen" title="上传 Sub2API 账号" width="wide" @close="uploadOpen = false">
      <div class="upload-dialog">
        <div class="upload-mode">
          <button :class="{ active: uploadMode === 'file' }" @click="uploadMode = 'file'"><Icon name="document" size="sm" />选择文件</button>
          <button :class="{ active: uploadMode === 'paste' }" @click="uploadMode = 'paste'"><Icon name="clipboard" size="sm" />粘贴 JSON</button>
          <button :class="{ active: uploadMode === 'oauth' }" @click="uploadMode = 'oauth'"><Icon name="key" size="sm" />OAuth 登录</button>
        </div>
        <label class="proxy-field">
          <span>账号代理</span>
          <select v-model="selectedProxyId" class="input">
            <option :value="null">不使用代理</option>
            <option v-for="proxy in proxies" :key="proxy.id" :value="proxy.id">
              {{ proxy.name }} · {{ proxy.protocol.toUpperCase() }}{{ proxy.expires_at ? ` · 有效期至 ${dateTime(proxy.expires_at)}` : '' }}
            </option>
          </select>
          <small>仅展示代理名称与协议，账号请求会自动通过所选代理转发。</small>
        </label>
        <label v-if="uploadMode === 'file'" class="file-drop">
          <input type="file" accept="application/json,.json" @change="selectFile" />
          <Icon name="upload" size="xl" />
          <strong>{{ uploadFileName || '选择 Sub2API JSON 文件' }}</strong>
          <span>支持单个账号、账号数组或 accounts 包装格式，最多 20 个</span>
        </label>
        <textarea v-else-if="uploadMode === 'paste'" v-model="uploadText" class="json-input" spellcheck="false" placeholder='{"name":"账号名称","platform":"openai","type":"oauth","credentials":{...}}'></textarea>
        <div v-else class="oauth-upload">
          <p>通过 OpenAI 官方 OAuth 登录，授权完成后把回调地址粘贴到下方，令牌会自动转换为账号并上传。</p>
          <button class="btn btn-secondary" :disabled="oauthLoading" @click="startOAuth">
            <Icon name="externalLink" size="sm" />{{ oauthUrl ? '重新打开授权页面' : '打开 OAuth 登录' }}
          </button>
          <a v-if="oauthUrl" class="oauth-url" :href="oauthUrl" target="_blank" rel="noopener">在新标签页打开授权页面</a>
          <label class="field-label">回调地址或授权码<textarea v-model.trim="oauthCallback" class="input oauth-input" placeholder="粘贴授权完成后的完整回调地址"></textarea></label>
          <label class="field-label">State（仅粘贴授权码时填写）<input v-model.trim="oauthStateInput" class="input" /></label>
          <button class="btn btn-primary" :disabled="oauthLoading || !oauthCallback" @click="completeOAuth">登录并读取账号</button>
          <small v-if="oauthEmail">已读取：{{ oauthEmail }}，点击“开始上传”后进入管理员配置的分组。</small>
        </div>
        <div v-if="uploadResults.length" class="upload-results">
          <div v-for="result in uploadResults" :key="result.index" :class="result.success ? 'ok' : 'failed'">
            <Icon :name="result.success ? 'checkCircle' : 'xCircle'" size="sm" />
            <span>{{ result.name || `第 ${result.index + 1} 个账号` }}</span>
            <strong>{{ result.success ? '上传成功' : result.error }}</strong>
          </div>
        </div>
      </div>
      <template #footer>
        <button class="btn btn-secondary" @click="uploadOpen = false">取消</button>
        <button v-if="uploadMode !== 'oauth'" class="btn btn-primary" :disabled="uploading || !uploadPayload" @click="submitUpload">
          <Icon v-if="uploading" name="refresh" size="sm" class="animate-spin" />{{ uploading ? '正在上传' : '开始上传' }}
        </button>
        <button v-else class="btn btn-primary" :disabled="uploading || !oauthPayload" @click="submitOAuthUpload">开始上传</button>
      </template>
    </BaseDialog>

    <BaseDialog :show="editOpen" title="编辑并发容量" @close="editOpen = false">
      <label class="field-label">并发容量<input v-model.number="editConcurrency" class="input" type="number" min="1" max="10000" /></label>
      <template #footer><button class="btn btn-secondary" @click="editOpen = false">取消</button><button class="btn btn-primary" @click="saveEdit">保存</button></template>
    </BaseDialog>

    <BaseDialog :show="withdrawOpen" title="申请提现" @close="withdrawOpen = false">
      <div class="withdraw-form">
        <div class="settlement-preview"><span>可提现 {{ money(summary.available) }}</span><strong>预计到账 {{ money(estimatedNet) }}</strong><small>平台抽佣 {{ number(config?.commission_percent || 0) }}%</small></div>
        <label class="field-label">收款方式<input v-model.trim="withdrawForm.payout_method" class="input" maxlength="32" placeholder="支付宝 / 微信 / 银行卡" /></label>
        <label class="field-label">收款信息<textarea v-model.trim="withdrawForm.payout_account" class="input payout-input" maxlength="2000" placeholder="填写收款账号、开户信息或收款说明"></textarea></label>
      </div>
      <template #footer><button class="btn btn-secondary" @click="withdrawOpen = false">取消</button><button class="btn btn-primary" :disabled="withdrawing" @click="submitWithdrawal">提交申请</button></template>
    </BaseDialog>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import AccountMarketQuotaCell from '@/components/accountmarket/AccountMarketQuotaCell.vue'
import accountMarketAPI, { type AccountMarketAccount, type AccountMarketConfig, type AccountMarketProxy, type AccountMarketSummary, type AccountMarketWithdrawal, type UploadResult } from '@/api/accountMarket'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'

const appStore = useAppStore()
const loading = ref(true)
const activeTab = ref<'accounts' | 'withdrawals'>('accounts')
const config = ref<AccountMarketConfig | null>(null)
const accounts = ref<AccountMarketAccount[]>([])
const withdrawals = ref<AccountMarketWithdrawal[]>([])
const summary = reactive<AccountMarketSummary>({ available: 0, locked: 0, withdrawn: 0, total: 0, account_count: 0 })
const uploadOpen = ref(false)
const uploadMode = ref<'file' | 'paste' | 'oauth'>('file')
const uploadText = ref('')
const uploadFileName = ref('')
const filePayload = ref<unknown>(null)
const uploadResults = ref<UploadResult[]>([])
const proxies = ref<AccountMarketProxy[]>([])
const proxiesLoaded = ref(false)
const selectedProxyId = ref<number | null>(null)
const uploading = ref(false)
const oauthLoading = ref(false)
const oauthUrl = ref('')
const oauthSessionId = ref('')
const oauthState = ref('')
const oauthStateInput = ref('')
const oauthCallback = ref('')
const oauthPayload = ref<Record<string, unknown> | null>(null)
const oauthEmail = ref('')
const busyId = ref<number | null>(null)
const editOpen = ref(false)
const editAccount = ref<AccountMarketAccount | null>(null)
const editConcurrency = ref(1)
const withdrawOpen = ref(false)
const withdrawing = ref(false)
const withdrawForm = reactive({ payout_method: '', payout_account: '' })
const uploadPayload = computed(() => uploadMode.value === 'file' ? filePayload.value : parseJSON(uploadText.value))
const estimatedNet = computed(() => summary.available * (1 - (config.value?.commission_percent || 0) / 100))

function parseJSON(value: string): unknown | null { try { return value.trim() ? JSON.parse(value) : null } catch { return null } }
function money(v: number) { return `$${Number(v || 0).toFixed(4)}` }
function number(v: number) { return Number(v || 0).toLocaleString(undefined, { maximumFractionDigits: 4 }) }
function dateTime(v?: string) { return v ? new Date(v).toLocaleString() : '-' }
function displayDate(v: unknown) {
  if (typeof v !== 'string' && typeof v !== 'number') return '-'
  const raw = typeof v === 'number' && v < 1e12 ? v * 1000 : v
  const date = new Date(raw)
  return Number.isNaN(date.getTime()) ? String(v) : date.toLocaleString()
}
function displayValue(v: unknown) {
  if (v === null || v === undefined || v === '') return '-'
  if (typeof v === 'number') return number(v)
  if (typeof v === 'object') {
    const record = v as Record<string, unknown>
    const candidate = record.remaining ?? record.value ?? record.balance ?? record.percent
    if (typeof candidate === 'number') return number(candidate)
    try { return JSON.stringify(v) } catch { return '-' }
  }
  return String(v)
}
function statusClass(a: AccountMarketAccount) { return a.account_status !== 'active' ? 'error' : a.schedulable ? 'active' : 'paused' }
function statusText(a: AccountMarketAccount) { return a.market_status === 'pending' ? '待验证' : a.market_status === 'rejected' ? '验证失败' : a.account_status !== 'active' ? '异常' : a.schedulable ? '运行中' : '已暂停' }
function withdrawalStatus(v: string) { return ({ pending: '待审核', approved: '已通过', paid: '已打款', rejected: '已拒绝', cancelled: '已取消' } as Record<string, string>)[v] || v }

async function loadAll() {
  loading.value = true
  try {
    const [cfg, accountItems, withdrawalItems] = await Promise.all([accountMarketAPI.getConfig(), accountMarketAPI.listAccounts(), accountMarketAPI.listWithdrawals()])
    const stats = await accountMarketAPI.getSummary()
    config.value = cfg; accounts.value = accountItems; withdrawals.value = withdrawalItems; Object.assign(summary, stats)
  } catch (error) { appStore.showError(extractApiErrorMessage(error, '账号收益数据加载失败')) }
  finally { loading.value = false }
}
async function openUpload() {
  uploadResults.value = []
  uploadOpen.value = true
  if (proxiesLoaded.value) return
  try { proxies.value = await accountMarketAPI.listProxies(); proxiesLoaded.value = true }
  catch (error) { appStore.showError(extractApiErrorMessage(error, '代理列表加载失败')) }
}
function openWithdrawal() {
  const latest = withdrawals.value.find(item => item.payout_account)
  if (latest) {
    withdrawForm.payout_method = latest.payout_method
    withdrawForm.payout_account = latest.payout_account
  }
  withdrawOpen.value = true
}
async function selectFile(event: Event) {
  const file = (event.target as HTMLInputElement).files?.[0]
  if (!file) return
  if (file.size > 2 * 1024 * 1024) { appStore.showError('文件不能超过 2 MB'); return }
  uploadFileName.value = file.name
  try { filePayload.value = JSON.parse(await file.text()) } catch { filePayload.value = null; appStore.showError('JSON 文件格式不正确') }
}
async function submitUpload() {
  if (!uploadPayload.value) { appStore.showError('请选择或填写有效的 JSON'); return }
  uploading.value = true
  try { uploadResults.value = await accountMarketAPI.upload(uploadPayload.value, selectedProxyId.value); if (uploadResults.value.some(i => i.success)) { appStore.showSuccess('账号上传完成'); await loadAll() } }
  catch (error) { appStore.showError(extractApiErrorMessage(error, '上传失败')) }
  finally { uploading.value = false }
}
async function startOAuth() {
  oauthLoading.value = true
  try {
    const result = await accountMarketAPI.generateOAuthURL(selectedProxyId.value)
    oauthUrl.value = result.auth_url
    oauthSessionId.value = result.session_id
    try { oauthState.value = new URL(result.auth_url).searchParams.get('state') || '' } catch { oauthState.value = '' }
    window.open(result.auth_url, '_blank', 'noopener,noreferrer')
  } catch (error) { appStore.showError(extractApiErrorMessage(error, 'OAuth 登录链接生成失败')) }
  finally { oauthLoading.value = false }
}
async function completeOAuth() {
  if (!oauthSessionId.value) { appStore.showError('请先打开 OAuth 登录'); return }
  oauthLoading.value = true
  try {
    let code = oauthCallback.value
    let state = oauthStateInput.value || oauthState.value
    try {
      const callback = new URL(oauthCallback.value)
      code = callback.searchParams.get('code') || ''
      state = callback.searchParams.get('state') || state
    } catch { /* input is a raw authorization code */ }
    if (!code || !state) { appStore.showError('回调地址中缺少 code 或 state'); return }
    const token = await accountMarketAPI.exchangeOAuthCode({ session_id: oauthSessionId.value, code, state, proxy_id: selectedProxyId.value || undefined })
    const credentials: Record<string, unknown> = { access_token: token.access_token, expires_at: token.expires_at }
    for (const key of ['refresh_token', 'id_token', 'email', 'chatgpt_account_id', 'chatgpt_user_id', 'organization_id', 'plan_type', 'client_id']) if (token[key as keyof typeof token]) credentials[key] = token[key as keyof typeof token]
    oauthEmail.value = token.email || ''
    oauthPayload.value = { name: token.email || 'OpenAI OAuth 账号', platform: 'openai', type: 'oauth', credentials, extra: { email: token.email, name: token.name } }
    appStore.showSuccess('OAuth 登录成功，请确认后上传')
  } catch (error) { appStore.showError(extractApiErrorMessage(error, 'OAuth 登录失败')) }
  finally { oauthLoading.value = false }
}
async function submitOAuthUpload() {
  if (!oauthPayload.value) return
  uploading.value = true
  try { uploadResults.value = await accountMarketAPI.upload(oauthPayload.value, selectedProxyId.value); if (uploadResults.value.some(i => i.success)) { appStore.showSuccess('OAuth 账号上传完成'); await loadAll() } }
  catch (error) { appStore.showError(extractApiErrorMessage(error, '账号上传失败')) }
  finally { uploading.value = false }
}
async function toggleSchedule(account: AccountMarketAccount) {
  busyId.value = account.account_id
  try { await accountMarketAPI.setSchedulable(account.account_id, !account.schedulable); account.schedulable = !account.schedulable; account.market_status = account.schedulable ? 'active' : 'paused' }
  catch (error) { appStore.showError(extractApiErrorMessage(error, '调度状态更新失败')) }
  finally { busyId.value = null }
}
function openEdit(account: AccountMarketAccount) { editAccount.value = account; editConcurrency.value = account.concurrency; editOpen.value = true }
async function saveEdit() {
  if (!editAccount.value) return
  try { const updated = await accountMarketAPI.updateAccount(editAccount.value.account_id, editConcurrency.value); Object.assign(editAccount.value, updated); editOpen.value = false; appStore.showSuccess('并发容量已更新') }
  catch (error) { appStore.showError(extractApiErrorMessage(error, '保存失败')) }
}
async function downloadAccount(account: AccountMarketAccount) {
  try { const data = await accountMarketAPI.exportAccount(account.account_id); const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' }); const url = URL.createObjectURL(blob); const anchor = document.createElement('a'); anchor.href = url; anchor.download = `${account.name || account.account_id}.json`; anchor.click(); URL.revokeObjectURL(url) }
  catch (error) { appStore.showError(extractApiErrorMessage(error, '导出失败')) }
}
async function remove(account: AccountMarketAccount) {
  if (!window.confirm(`确认移除账号“${account.name}”？历史收益不会丢失。`)) return
  try { await accountMarketAPI.removeAccount(account.account_id); await loadAll(); appStore.showSuccess('账号已移除') }
  catch (error) { appStore.showError(extractApiErrorMessage(error, '移除失败')) }
}
async function submitWithdrawal() {
  withdrawing.value = true
  try { await accountMarketAPI.requestWithdrawal(withdrawForm); withdrawOpen.value = false; await loadAll(); appStore.showSuccess('提现申请已提交') }
  catch (error) { appStore.showError(extractApiErrorMessage(error, '提现申请失败')) }
  finally { withdrawing.value = false }
}
async function cancel(id: number) { try { await accountMarketAPI.cancelWithdrawal(id); await loadAll(); appStore.showSuccess('提现申请已取消') } catch (error) { appStore.showError(extractApiErrorMessage(error, '取消失败')) } }
onMounted(loadAll)
</script>

<style scoped>
.market-page{max-width:1480px;margin:0 auto;display:flex;flex-direction:column;gap:20px}.page-heading{display:flex;justify-content:space-between;align-items:flex-end;gap:20px}.page-heading h1{font-size:28px;font-weight:750;color:rgb(17 24 39);margin:2px 0 4px}.dark .page-heading h1{color:white}.page-heading p{color:rgb(107 114 128);font-size:14px}.eyebrow{font-size:11px!important;font-weight:700;letter-spacing:.12em;color:rgb(5 150 105)!important}.notice{display:flex;align-items:center;gap:8px;padding:11px 14px;border:1px solid rgb(253 230 138);background:rgb(255 251 235);color:rgb(146 64 14);border-radius:6px;font-size:13px}.summary-strip{display:grid;grid-template-columns:minmax(260px,1.4fr) repeat(4,minmax(130px,1fr));border:1px solid rgb(229 231 235);background:white;border-radius:8px;overflow:hidden}.dark .summary-strip,.dark .panel{background:rgb(17 24 39);border-color:rgb(55 65 81)}.summary-strip>div{padding:18px 20px;border-right:1px solid rgb(229 231 235);display:flex;flex-direction:column;gap:5px}.dark .summary-strip>div{border-color:rgb(55 65 81)}.summary-strip>div:last-child{border:0}.summary-strip span{font-size:12px;color:rgb(107 114 128)}.summary-strip strong{font-size:20px;color:rgb(17 24 39)}.dark .summary-strip strong{color:white}.summary-primary{position:relative;background:rgb(236 253 245)}.dark .summary-primary{background:rgb(6 78 59 / .2)}.summary-primary strong{color:rgb(5 150 105)!important;font-size:27px!important}.summary-primary .btn{position:absolute;right:18px;bottom:18px}.view-tabs{display:flex;border-bottom:1px solid rgb(229 231 235);gap:24px}.view-tabs button{padding:9px 2px;border-bottom:2px solid transparent;color:rgb(107 114 128);font-size:14px}.view-tabs button.active{border-color:rgb(5 150 105);color:rgb(5 150 105);font-weight:650}.panel{background:white;border:1px solid rgb(229 231 235);border-radius:8px;overflow:hidden}.panel-head{padding:18px 20px;border-bottom:1px solid rgb(229 231 235);display:flex;justify-content:space-between;align-items:center}.dark .panel-head{border-color:rgb(55 65 81)}.panel-head h2{font-size:16px;font-weight:700;color:rgb(17 24 39)}.dark .panel-head h2{color:white}.panel-head p{font-size:12px;color:rgb(107 114 128);margin-top:3px}.icon-action{width:36px;height:36px;display:grid;place-items:center;border:1px solid rgb(229 231 235);border-radius:6px;color:rgb(75 85 99)}.empty-state{min-height:220px;display:flex;flex-direction:column;justify-content:center;align-items:center;gap:9px;color:rgb(156 163 175);font-size:13px}.empty-state strong{color:rgb(75 85 99);font-size:15px}.account-list{display:flex;flex-direction:column}.account-row{display:grid;grid-template-columns:minmax(260px,1.2fr) minmax(580px,2fr) auto;align-items:center;gap:20px;padding:18px 20px;border-bottom:1px solid rgb(243 244 246)}.dark .account-row{border-color:rgb(31 41 55)}.account-row:last-child{border:0}.account-identity{display:flex;align-items:center;gap:12px;min-width:0}.platform-mark{width:38px;height:38px;display:grid;place-items:center;border-radius:6px;background:rgb(17 24 39);color:white;font-size:11px;font-weight:800;letter-spacing:.04em}.identity-line{display:flex;align-items:center;gap:8px}.identity-line strong{color:rgb(17 24 39);font-size:14px;white-space:nowrap;overflow:hidden;text-overflow:ellipsis}.dark .identity-line strong{color:white}.account-identity p{font-size:12px;color:rgb(107 114 128);margin-top:3px}.status-dot{font-size:11px;padding:2px 6px;border-radius:999px;background:rgb(243 244 246);color:rgb(75 85 99)}.status-dot.active{background:rgb(209 250 229);color:rgb(4 120 87)}.status-dot.paused{background:rgb(254 243 199);color:rgb(180 83 9)}.status-dot.error{background:rgb(254 226 226);color:rgb(185 28 28)}.account-metrics{display:grid;grid-template-columns:repeat(6,minmax(78px,1fr));gap:14px}.account-metrics div{min-width:0}.account-metrics span{display:block;font-size:11px;color:rgb(156 163 175);margin-bottom:4px}.account-metrics strong{display:block;font-size:12px;color:rgb(55 65 81);overflow:hidden;text-overflow:ellipsis;white-space:nowrap}.dark .account-metrics strong{color:rgb(229 231 235)}.earning{color:rgb(5 150 105)!important;font-weight:700}.account-actions{display:flex;align-items:center;justify-content:flex-end;gap:7px}.switch-control{display:flex;align-items:center;gap:6px;font-size:12px;color:rgb(75 85 99);white-space:nowrap}.switch-control input{display:none}.switch-control span{width:30px;height:17px;background:rgb(209 213 219);border-radius:999px;position:relative;transition:.2s}.switch-control span:after{content:'';position:absolute;width:13px;height:13px;left:2px;top:2px;background:white;border-radius:50%;transition:.2s}.switch-control input:checked+span{background:rgb(5 150 105)}.switch-control input:checked+span:after{transform:translateX(13px)}.account-error{grid-column:1/-1;color:rgb(185 28 28);font-size:12px;background:rgb(254 242 242);padding:7px 10px;border-radius:4px}.table-wrap{overflow:auto}table{width:100%;min-width:920px;border-collapse:collapse;text-align:left;font-size:13px}th{padding:11px 16px;background:rgb(249 250 251);color:rgb(107 114 128);font-weight:600;border-bottom:1px solid rgb(229 231 235)}td{padding:13px 16px;border-bottom:1px solid rgb(243 244 246);color:rgb(55 65 81)}.dark th{background:rgb(31 41 55);border-color:rgb(55 65 81)}.dark td{color:rgb(209 213 219);border-color:rgb(31 41 55)}td small{display:block;color:rgb(185 28 28);margin-top:4px}.payout{max-width:240px;display:block;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}.withdraw-status{display:inline-flex;padding:3px 7px;border-radius:999px;background:rgb(243 244 246);font-size:11px}.withdraw-status.paid{background:rgb(209 250 229);color:rgb(4 120 87)}.withdraw-status.pending{background:rgb(254 243 199);color:rgb(180 83 9)}.withdraw-status.rejected{background:rgb(254 226 226);color:rgb(185 28 28)}.upload-dialog{display:flex;flex-direction:column;gap:16px}.upload-mode{display:flex;padding:3px;background:rgb(243 244 246);border-radius:6px;width:max-content}.upload-mode button{display:flex;align-items:center;gap:6px;padding:7px 12px;border-radius:4px;font-size:13px;color:rgb(107 114 128)}.upload-mode button.active{background:white;color:rgb(17 24 39);box-shadow:0 1px 2px rgb(0 0 0 / .08)}.file-drop{height:230px;border:1px dashed rgb(156 163 175);border-radius:7px;display:flex;flex-direction:column;align-items:center;justify-content:center;gap:8px;color:rgb(107 114 128);cursor:pointer;background:rgb(249 250 251)}.file-drop:hover{border-color:rgb(5 150 105);background:rgb(236 253 245)}.file-drop input{display:none}.file-drop strong{color:rgb(31 41 55)}.file-drop span{font-size:12px}.json-input{height:230px;width:100%;resize:vertical;border:1px solid rgb(209 213 219);border-radius:6px;padding:12px;font:12px/1.6 ui-monospace,monospace;background:rgb(17 24 39);color:rgb(229 231 235)}.upload-results{max-height:160px;overflow:auto;display:flex;flex-direction:column;gap:6px}.upload-results div{display:grid;grid-template-columns:20px 1fr auto;gap:7px;align-items:center;padding:8px 10px;border-radius:5px;font-size:12px}.upload-results .ok{background:rgb(236 253 245);color:rgb(4 120 87)}.upload-results .failed{background:rgb(254 242 242);color:rgb(185 28 28)}.field-label{display:flex;flex-direction:column;gap:7px;font-size:13px;color:rgb(55 65 81)}.withdraw-form{display:flex;flex-direction:column;gap:16px}.settlement-preview{display:grid;grid-template-columns:1fr auto;gap:4px;padding:14px;background:rgb(236 253 245);border:1px solid rgb(167 243 208);border-radius:6px}.settlement-preview span,.settlement-preview small{font-size:12px;color:rgb(5 150 105)}.settlement-preview strong{font-size:17px;color:rgb(4 120 87)}.settlement-preview small{grid-column:1/-1}.payout-input{min-height:90px;resize:vertical}
@media(max-width:1200px){.summary-strip{grid-template-columns:repeat(2,1fr)}.summary-strip>div{border-bottom:1px solid rgb(229 231 235)}.summary-primary{grid-column:1/-1}.account-row{grid-template-columns:1fr}.account-actions{justify-content:flex-start}.account-metrics{grid-template-columns:repeat(3,1fr)}}@media(max-width:640px){.page-heading{align-items:flex-start;flex-direction:column}.page-heading .btn{width:100%}.summary-strip{grid-template-columns:1fr 1fr}.summary-strip>div{padding:14px}.account-metrics{grid-template-columns:repeat(2,1fr)}.account-actions{display:grid;grid-template-columns:1fr 1fr}.switch-control{grid-column:1/-1}.summary-primary .btn{position:static;margin-top:7px}.summary-primary{display:flex!important}}
.proxy-field{display:flex;flex-direction:column;gap:7px;padding:12px 14px;border:1px solid rgb(229 231 235);border-radius:7px;background:rgb(249 250 251);font-size:13px;font-weight:600;color:rgb(55 65 81)}
.proxy-field small{font-size:11px;font-weight:400;color:rgb(107 114 128)}
.proxy-field select{width:100%;background:white}
.dark .proxy-field{border-color:rgb(55 65 81);background:rgb(31 41 55);color:rgb(229 231 235)}
.dark .proxy-field select{background:rgb(17 24 39)}
.oauth-upload{display:flex;flex-direction:column;gap:12px;padding:16px;border:1px solid rgb(209 250 229);border-radius:7px;background:rgb(240 253 250)}
.oauth-upload p,.oauth-upload small{font-size:12px;color:rgb(55 65 81);line-height:1.6}
.oauth-url{font-size:12px;color:rgb(5 150 105);text-decoration:underline}
.oauth-input{min-height:82px;resize:vertical}
.dark .oauth-upload{border-color:rgb(6 78 59);background:rgb(6 78 59 / .2)}
.dark .oauth-upload p,.dark .oauth-upload small{color:rgb(209 250 229)}
.account-metrics .quota-cell{grid-column:1/-1}
</style>
