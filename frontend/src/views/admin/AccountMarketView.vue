<template>
  <AppLayout>
    <div class="admin-market">
      <header class="page-heading">
        <div><p class="eyebrow">CONTRIBUTOR OPERATIONS</p><h1>账号收益管理</h1><p>配置账号上传入口、提现抽佣，并完成提现审核闭环。</p></div>
        <button class="btn btn-primary" :disabled="saving" @click="saveConfig"><Icon name="check" size="sm" />保存配置</button>
      </header>

      <section class="settings-band">
        <label class="toggle-setting"><span><strong>开放账号上传</strong><small>关闭后不接受新账号，已有账号继续运行。</small></span><input v-model="form.enabled" type="checkbox" /><i></i></label>
        <label><span>统一上传分组</span><select v-model="form.target_group_id" class="input"><option :value="null">请选择分组</option><option v-for="group in groups" :key="group.id" :value="group.id">{{ group.name }} · {{ group.platform }}</option></select></label>
        <label><span>提现抽佣比例</span><div class="suffix-input"><input v-model.number="form.commission_percent" class="input" type="number" min="0" max="100" step="0.1" /><b>%</b></div></label>
        <label><span>最低提现金额</span><div class="suffix-input"><input v-model.number="form.minimum_withdrawal" class="input" type="number" min="0" step="0.01" /><b>$</b></div></label>
        <label class="toggle-setting"><span><strong>提现需要审核</strong><small>关闭后申请直接进入已通过状态，仍需标记打款。</small></span><input v-model="form.withdrawal_review_required" type="checkbox" /><i></i></label>
      </section>

      <div class="tabs"><button :class="{active:tab==='withdrawals'}" @click="tab='withdrawals'">提现审核 <span>{{ pendingCount }}</span></button><button :class="{active:tab==='accounts'}" @click="tab='accounts'">贡献账号 <span>{{ accounts.length }}</span></button></div>

      <section class="table-panel">
        <div class="table-head"><div><h2>{{ tab === 'withdrawals' ? '提现申请' : '贡献账号' }}</h2><p>{{ tab === 'withdrawals' ? '审核通过后，在实际完成转账时标记已打款。' : '仅展示通过账号收益入口上传的账号。' }}</p></div><button class="icon-action" title="刷新" @click="load"><Icon name="refresh" :class="{'animate-spin':loading}" /></button></div>
        <div v-if="loading" class="empty">正在读取数据...</div>
        <div v-else-if="tab==='withdrawals'" class="table-wrap">
          <table><thead><tr><th>用户</th><th>申请时间</th><th>申请金额</th><th>抽佣</th><th>实际到账</th><th>收款信息</th><th>状态</th><th>操作</th></tr></thead>
            <tbody><tr v-for="item in withdrawals" :key="item.id"><td>{{ item.user_email || `用户 #${item.user_id}` }}</td><td>{{ dateTime(item.created_at) }}</td><td>{{ money(item.gross_amount) }}</td><td>{{ money(item.commission_amount) }}<small>{{ number(item.commission_percent) }}%</small></td><td class="earning">{{ money(item.net_amount) }}</td><td><span class="payout">{{ item.payout_method }} · {{ item.payout_account }}</span></td><td><span class="status" :class="item.status">{{ statusText(item.status) }}</span><small v-if="item.review_note">{{ item.review_note }}</small></td><td><div class="actions"><button v-if="item.status==='pending'" class="btn btn-secondary btn-sm" @click="review(item,'approved')">通过</button><button v-if="item.status==='pending'||item.status==='approved'" class="btn btn-primary btn-sm" @click="review(item,'paid')">已打款</button><button v-if="item.status==='pending'||item.status==='approved'" class="btn btn-danger btn-sm" @click="reject(item)">拒绝</button></div></td></tr><tr v-if="withdrawals.length===0"><td colspan="8" class="empty">暂无提现记录</td></tr></tbody>
          </table>
        </div>
        <div v-else class="table-wrap">
          <table><thead><tr><th>账号</th><th>平台</th><th>状态</th><th>分组</th><th>并发</th><th>账号倍率</th><th>分组倍率</th><th>累计收益</th><th>最近使用</th></tr></thead>
            <tbody><tr v-for="item in accounts" :key="item.account_id"><td><strong>{{ item.name }}</strong><small>{{ item.email || '未识别邮箱' }} · #{{ item.account_id }}</small><small>{{ item.uploader_email || `上传用户 #${item.uploader_user_id}` }}</small></td><td>{{ item.platform }} / {{ item.type }}</td><td><span class="status" :class="item.schedulable?'paid':'cancelled'">{{ item.account_status }} · {{ item.schedulable ? '调度中' : '已暂停' }}</span></td><td>{{ item.group_name || '-' }}</td><td>{{ item.concurrency }}</td><td>{{ number(item.rate_multiplier) }}x</td><td>{{ number(item.group_rate_multiplier) }}x</td><td class="earning">{{ money(item.total_earned) }}</td><td>{{ dateTime(item.last_used_at) }}</td></tr><tr v-if="accounts.length===0"><td colspan="9" class="empty">暂无贡献账号</td></tr></tbody>
          </table>
        </div>
      </section>
    </div>

    <BaseDialog :show="rejectOpen" title="拒绝提现" @close="rejectOpen=false"><label class="field-label">拒绝原因<textarea v-model.trim="rejectNote" class="input note-input" maxlength="1000" placeholder="用户可以在提现记录中看到该原因"></textarea></label><template #footer><button class="btn btn-secondary" @click="rejectOpen=false">取消</button><button class="btn btn-danger" @click="confirmReject">确认拒绝</button></template></BaseDialog>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import adminAccountMarketAPI from '@/api/admin/accountMarket'
import * as groupsAPI from '@/api/admin/groups'
import type { AdminGroup } from '@/types'
import type { AccountMarketAccount, AccountMarketConfig, AccountMarketWithdrawal } from '@/api/accountMarket'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'

const appStore=useAppStore(); const loading=ref(true); const saving=ref(false); const tab=ref<'withdrawals'|'accounts'>('withdrawals')
const groups=ref<AdminGroup[]>([]); const accounts=ref<AccountMarketAccount[]>([]); const withdrawals=ref<AccountMarketWithdrawal[]>([])
const form=reactive<AccountMarketConfig>({enabled:false,target_group_id:null,commission_percent:0,minimum_withdrawal:0,withdrawal_review_required:true})
const rejectOpen=ref(false); const rejectNote=ref(''); const rejecting=ref<AccountMarketWithdrawal|null>(null)
const pendingCount=computed(()=>withdrawals.value.filter(i=>i.status==='pending').length)
function money(v:number){return `$${Number(v||0).toFixed(4)}`} function number(v:number){return Number(v||0).toLocaleString(undefined,{maximumFractionDigits:4})} function dateTime(v?:string){return v?new Date(v).toLocaleString():'-'}
function statusText(v:string){return ({pending:'待审核',approved:'已通过',paid:'已打款',rejected:'已拒绝',cancelled:'已取消'} as Record<string,string>)[v]||v}
async function load(){loading.value=true;try{const [cfg,groupItems,accountItems,withdrawalItems]=await Promise.all([adminAccountMarketAPI.getConfig(),groupsAPI.getAllIncludingInactive(),adminAccountMarketAPI.listAccounts(),adminAccountMarketAPI.listWithdrawals()]);Object.assign(form,cfg);groups.value=groupItems;accounts.value=accountItems;withdrawals.value=withdrawalItems}catch(error){appStore.showError(extractApiErrorMessage(error,'账号收益管理数据加载失败'))}finally{loading.value=false}}
async function saveConfig(){saving.value=true;try{const updated=await adminAccountMarketAPI.updateConfig({...form});Object.assign(form,updated);appStore.showSuccess('账号收益配置已保存')}catch(error){appStore.showError(extractApiErrorMessage(error,'配置保存失败'))}finally{saving.value=false}}
async function review(item:AccountMarketWithdrawal,status:'approved'|'paid'){const action=status==='paid'?'确认款项已经实际转出？':'确认通过这笔提现？';if(!window.confirm(action))return;try{await adminAccountMarketAPI.reviewWithdrawal(item.id,status);await load();appStore.showSuccess(status==='paid'?'已标记打款':'提现已通过')}catch(error){appStore.showError(extractApiErrorMessage(error,'操作失败'))}}
function reject(item:AccountMarketWithdrawal){rejecting.value=item;rejectNote.value='';rejectOpen.value=true}
async function confirmReject(){if(!rejecting.value)return;try{await adminAccountMarketAPI.reviewWithdrawal(rejecting.value.id,'rejected',rejectNote.value);rejectOpen.value=false;await load();appStore.showSuccess('提现已拒绝，收益已退回用户可提现余额')}catch(error){appStore.showError(extractApiErrorMessage(error,'拒绝失败'))}}
onMounted(load)
</script>

<style scoped>
.admin-market{max-width:1500px;margin:0 auto;display:flex;flex-direction:column;gap:20px}.page-heading{display:flex;justify-content:space-between;align-items:flex-end;gap:20px}.page-heading h1{font-size:28px;font-weight:750;color:rgb(17 24 39);margin:2px 0 4px}.dark .page-heading h1{color:white}.page-heading p{font-size:14px;color:rgb(107 114 128)}.eyebrow{font-size:11px!important;font-weight:700;letter-spacing:.12em;color:rgb(5 150 105)!important}.settings-band{display:grid;grid-template-columns:1.3fr 1.5fr .8fr .8fr 1.3fr;border:1px solid rgb(229 231 235);background:white;border-radius:8px;overflow:hidden}.dark .settings-band,.dark .table-panel{background:rgb(17 24 39);border-color:rgb(55 65 81)}.settings-band>label{padding:16px;border-right:1px solid rgb(229 231 235);display:flex;flex-direction:column;gap:7px;font-size:12px;color:rgb(75 85 99)}.dark .settings-band>label{border-color:rgb(55 65 81);color:rgb(156 163 175)}.settings-band>label:last-child{border:0}.toggle-setting{position:relative;flex-direction:row!important;align-items:center;justify-content:space-between}.toggle-setting span{display:flex;flex-direction:column;gap:3px}.toggle-setting strong{font-size:13px;color:rgb(31 41 55)}.dark .toggle-setting strong{color:white}.toggle-setting small{color:rgb(107 114 128);line-height:1.3}.toggle-setting input{display:none}.toggle-setting i{width:34px;height:19px;border-radius:999px;background:rgb(209 213 219);position:relative;flex:0 0 auto}.toggle-setting i:after{content:'';position:absolute;width:15px;height:15px;left:2px;top:2px;background:white;border-radius:50%;transition:.2s}.toggle-setting input:checked+i{background:rgb(5 150 105)}.toggle-setting input:checked+i:after{transform:translateX(15px)}.suffix-input{position:relative}.suffix-input b{position:absolute;right:10px;top:50%;transform:translateY(-50%);color:rgb(156 163 175)}.suffix-input .input{padding-right:28px}.tabs{display:flex;gap:24px;border-bottom:1px solid rgb(229 231 235)}.tabs button{padding:9px 2px;border-bottom:2px solid transparent;font-size:14px;color:rgb(107 114 128)}.tabs button.active{border-color:rgb(5 150 105);color:rgb(5 150 105);font-weight:650}.tabs span{margin-left:4px;padding:1px 5px;border-radius:999px;background:rgb(243 244 246);font-size:10px}.table-panel{border:1px solid rgb(229 231 235);background:white;border-radius:8px;overflow:hidden}.table-head{padding:16px 18px;display:flex;justify-content:space-between;align-items:center;border-bottom:1px solid rgb(229 231 235)}.dark .table-head{border-color:rgb(55 65 81)}.table-head h2{font-size:16px;font-weight:700;color:rgb(17 24 39)}.dark .table-head h2{color:white}.table-head p{font-size:12px;color:rgb(107 114 128);margin-top:3px}.icon-action{width:36px;height:36px;display:grid;place-items:center;border:1px solid rgb(229 231 235);border-radius:6px;color:rgb(75 85 99)}.table-wrap{overflow:auto}table{width:100%;min-width:1050px;border-collapse:collapse;text-align:left;font-size:12px}th{padding:10px 14px;background:rgb(249 250 251);color:rgb(107 114 128);font-weight:600;border-bottom:1px solid rgb(229 231 235)}td{padding:12px 14px;border-bottom:1px solid rgb(243 244 246);color:rgb(55 65 81);vertical-align:middle}.dark th{background:rgb(31 41 55);border-color:rgb(55 65 81)}.dark td{color:rgb(209 213 219);border-color:rgb(31 41 55)}td strong{display:block;color:rgb(31 41 55)}.dark td strong{color:white}td small{display:block;color:rgb(107 114 128);margin-top:3px}.earning{color:rgb(5 150 105)!important;font-weight:700}.payout{display:block;max-width:220px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}.status{display:inline-flex;padding:3px 7px;border-radius:999px;background:rgb(243 244 246);font-size:11px}.status.pending{background:rgb(254 243 199);color:rgb(180 83 9)}.status.approved{background:rgb(219 234 254);color:rgb(29 78 216)}.status.paid{background:rgb(209 250 229);color:rgb(4 120 87)}.status.rejected{background:rgb(254 226 226);color:rgb(185 28 28)}.actions{display:flex;gap:5px;white-space:nowrap}.empty{text-align:center!important;padding:70px 20px!important;color:rgb(156 163 175)!important}.field-label{display:flex;flex-direction:column;gap:7px;font-size:13px;color:rgb(55 65 81)}.note-input{min-height:100px;resize:vertical}@media(max-width:1100px){.settings-band{grid-template-columns:1fr 1fr}.settings-band>label{border-bottom:1px solid rgb(229 231 235)}}@media(max-width:640px){.page-heading{align-items:flex-start;flex-direction:column}.page-heading .btn{width:100%}.settings-band{grid-template-columns:1fr}.settings-band>label{border-right:0}}
</style>
