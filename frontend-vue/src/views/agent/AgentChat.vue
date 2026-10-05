<template>
  <PageContainer title="AI 研究助手" description="调用真实行情、量化指标、公告资讯、持仓、回测与知识库的只读研究对话">
    <template #actions><el-button @click="createSession">新建会话</el-button><el-button @click="$router.push('/system/models')">AI 模型设置</el-button></template>
    <el-alert title="助手只能读取研究数据，不能下单、修改账户或启停自动交易。所有内容仅供研究，不构成投资建议。" type="warning" show-icon :closable="false" class="chat-notice"/>
    <section class="chat-shell">
      <aside class="session-panel">
        <div class="panel-title"><strong>研究会话</strong><span>{{sessions.length}} 个</span></div>
        <el-scrollbar class="session-list">
          <button v-for="item in sessions" :key="item.id" class="session-item" :class="{active:item.id===sessionId}" @click="selectSession(item.id)">
            <span class="session-title">{{item.title}}</span><small>{{formatTime(item.last_message_at||item.created_at)}}</small>
          </button>
        </el-scrollbar>
        <div v-if="current" class="session-actions"><el-button size="small" @click="renameSession">重命名</el-button><el-button size="small" @click="clearMessages">清空</el-button><el-button size="small" type="danger" plain @click="deleteSession">删除</el-button></div>
      </aside>

      <div class="conversation-panel">
        <div class="conversation-head"><div><el-button class="session-mobile" circle aria-label="打开研究会话" @click="sessionDrawer=true"><el-icon><Menu /></el-icon></el-button><strong>{{current?.title||'研究会话'}}</strong><span :class="connected?'online':'offline'">{{connected?'已连接':'连接中'}}</span></div><el-button class="evidence-mobile" @click="evidenceDrawer=true">证据</el-button></div>
        <el-scrollbar ref="messageScroll" class="message-list">
          <el-empty v-if="!messages.length&&!loading" description="输入股票代码、公司名称或研究问题开始分析"/>
          <article v-for="item in messages" :key="`${item.request_id}-${item.role}-${item.id}`" class="message" :class="item.role" @click="selectedMessage=item">
            <header><strong>{{item.role==='user'?'你':'AI 研究助手'}}</strong><span>{{formatTime(item.created_at)}}</span></header>
            <div class="message-content">{{item.content}}</div>
            <footer v-if="item.role==='assistant'"><el-tag v-if="item.status==='FAILED'" type="danger">分析失败</el-tag><span v-else>{{item.provider||'本地量化'}} {{item.model?`/ ${item.model}`:''}} · {{item.latency_ms||0}} ms</span></footer>
          </article>
          <div v-if="thinking" class="thinking"><span></span><span></span><span></span>正在查询真实数据并生成研究报告</div>
        </el-scrollbar>
        <div class="composer">
          <div class="context-row"><el-input v-model="symbol" maxlength="6" clearable placeholder="股票代码（可选）"/><span>支持查询行情、K线、指标、公告、资讯、持仓、回测和知识库</span></div>
          <el-input v-model="draft" type="textarea" :rows="3" maxlength="4000" show-word-limit resize="none" placeholder="例如：结合近期公告、量价、MACD 和资金流分析中国巨石，说明主要风险。" @keydown.ctrl.enter.prevent="sendMessage"/>
          <div class="send-row"><span>Ctrl + Enter 发送</span><el-button type="primary" :loading="thinking" :disabled="!draft.trim()||!sessionId||!connected" @click="sendMessage">发送研究问题</el-button></div>
        </div>
      </div>

      <aside class="evidence-panel"><EvidenceContent :message="selectedMessage"/></aside>
    </section>
    <el-drawer v-model="sessionDrawer" title="研究会话" direction="ltr" size="min(340px,88vw)" class="session-drawer">
      <div class="drawer-session-list">
        <button v-for="item in sessions" :key="item.id" class="session-item" :class="{active:item.id===sessionId}" @click="selectSession(item.id);sessionDrawer=false">
          <span class="session-title">{{item.title}}</span><small>{{formatTime(item.last_message_at||item.created_at)}}</small>
        </button>
      </div>
      <div v-if="current" class="drawer-session-actions"><el-button @click="renameSession">重命名</el-button><el-button @click="clearMessages">清空</el-button><el-button type="danger" plain @click="deleteSession">删除</el-button></div>
    </el-drawer>
    <el-drawer v-model="evidenceDrawer" title="证据与工具调用" size="min(420px,92vw)"><EvidenceContent :message="selectedMessage"/></el-drawer>
  </PageContainer>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Menu } from '@element-plus/icons-vue'
import PageContainer from '@/components/common/PageContainer.vue'
import EvidenceContent from '@/components/agent/EvidenceContent.vue'
import { agentApi } from '@/api/agent'
import { storage } from '@/utils/storage'
import { formatTime } from '@/utils/time'
import type { AgentChatMessage, AgentChatSession } from '@/types/agent'

type ChatEvent={type:string;trace_id?:string;data?:Record<string,unknown>}
const sessions=ref<AgentChatSession[]>([]),messages=ref<AgentChatMessage[]>([]),sessionId=ref(0),loading=ref(false),thinking=ref(false),connected=ref(false),draft=ref(''),symbol=ref(''),selectedMessage=ref<AgentChatMessage>(),sessionDrawer=ref(false),evidenceDrawer=ref(false),messageScroll=ref<{setScrollTop:(value:number)=>void}>()
let socket:WebSocket|null=null,reconnectTimer:number|undefined,activeRequest=''
const current=computed(()=>sessions.value.find(v=>v.id===sessionId.value))
const wsURL=()=>{const token=encodeURIComponent(storage.getToken());const protocol=location.protocol==='https:'?'wss:':'ws:';return `${protocol}//${location.host}/ws/agent-chat?token=${token}`}
const loadSessions=async()=>{sessions.value=await agentApi.chatSessions();if(!sessions.value.length){const item=await agentApi.createChatSession();sessions.value=[item]}if(!sessions.value.some(v=>v.id===sessionId.value))sessionId.value=sessions.value[0].id;await loadMessages()}
const loadMessages=async()=>{if(!sessionId.value)return;loading.value=true;try{messages.value=await agentApi.chatMessages(sessionId.value);selectedMessage.value=[...messages.value].reverse().find(v=>v.role==='assistant')}finally{loading.value=false;scrollBottom()}}
const selectSession=async(id:number)=>{if(thinking.value)return ElMessage.warning('当前分析尚未完成，请稍候');sessionId.value=id;await loadMessages()}
const createSession=async()=>{const item=await agentApi.createChatSession();sessions.value.unshift(item);sessionId.value=item.id;messages.value=[];selectedMessage.value=undefined}
const renameSession=async()=>{if(!current.value)return;const result=await ElMessageBox.prompt('输入便于识别的研究会话名称','重命名会话',{inputValue:current.value.title,inputPattern:/\S+/,inputErrorMessage:'名称不能为空'});await agentApi.renameChatSession(sessionId.value,result.value);await loadSessions()}
const deleteSession=async()=>{await ElMessageBox.confirm('删除会话后将从列表隐藏，确认继续？','删除研究会话',{type:'warning'});await agentApi.deleteChatSession(sessionId.value);sessionId.value=0;await loadSessions()}
const clearMessages=async()=>{await ElMessageBox.confirm('确认清空当前会话的全部消息？','清空聊天记录',{type:'warning'});await agentApi.clearChatMessages(sessionId.value);messages.value=[];selectedMessage.value=undefined;ElMessage.success('聊天记录已清空')}
const connect=()=>{if(socket?.readyState===WebSocket.OPEN)return;socket=new WebSocket(wsURL());socket.onopen=()=>connected.value=true;socket.onmessage=handleEvent;socket.onerror=()=>connected.value=false;socket.onclose=()=>{connected.value=false;if(storage.getToken())reconnectTimer=window.setTimeout(connect,2500)}}
const handleEvent=(event:MessageEvent)=>{const payload=JSON.parse(String(event.data)) as ChatEvent;const data=payload.data||{};if(payload.type==='chat.delta'&&String(data.request_id||'')===activeRequest){const assistant=messages.value.find(v=>v.request_id===activeRequest&&v.role==='assistant');if(assistant)assistant.content+=String(data.content||'');scrollBottom()}else if(payload.type==='chat.completed'){thinking.value=false;const saved=data.message as unknown as AgentChatMessage;const index=messages.value.findIndex(v=>v.request_id===saved.request_id&&v.role==='assistant');if(index>=0)messages.value[index]=saved;else messages.value.push(saved);selectedMessage.value=saved;void loadSessions()}else if(payload.type==='chat.error'){thinking.value=false;const message=String(data.message||'智能研究失败，请稍后重试');const trace=payload.trace_id?`trace_id=${payload.trace_id}`:'';const assistant=messages.value.find(v=>v.request_id===String(data.request_id||activeRequest)&&v.role==='assistant');if(assistant){assistant.status='FAILED';assistant.content=`${message}${trace?`\n${trace}`:''}`;assistant.error_message=message;selectedMessage.value=assistant}ElMessage.error(`${message} ${trace}`);void loadSessions()}}
const sendMessage=()=>{const content=draft.value.trim();if(!content||!sessionId.value||!socket||socket.readyState!==WebSocket.OPEN)return;activeRequest=crypto.randomUUID();messages.value.push({id:0,session_id:sessionId.value,request_id:activeRequest,role:'user',content,status:'COMPLETED',provider:'',model:'',input_tokens:0,output_tokens:0,latency_ms:0,citations_json:'[]',tool_calls_json:'[]',error_message:'',created_at:new Date().toISOString()},{id:0,session_id:sessionId.value,request_id:activeRequest,role:'assistant',content:'',status:'STREAMING',provider:'',model:'',input_tokens:0,output_tokens:0,latency_ms:0,citations_json:'[]',tool_calls_json:'[]',error_message:'',created_at:new Date().toISOString()});thinking.value=true;socket.send(JSON.stringify({type:'chat.ask',request_id:activeRequest,session_id:sessionId.value,content,symbol:symbol.value.trim()}));draft.value='';scrollBottom()}
const scrollBottom=()=>nextTick(()=>messageScroll.value?.setScrollTop(999999))
onMounted(async()=>{await loadSessions();connect()});onBeforeUnmount(()=>{if(reconnectTimer)clearTimeout(reconnectTimer);socket?.close();socket=null})
</script>

<style scoped>
.chat-notice{margin-bottom:14px}.chat-shell{display:grid;grid-template-columns:230px minmax(0,1fr) 300px;height:calc(100dvh - 210px);min-height:590px;border:1px solid var(--qs-border);background:#fff}.session-panel,.evidence-panel{min-width:0;background:#f8fafb}.session-panel{display:flex;flex-direction:column;border-right:1px solid var(--qs-border)}.evidence-panel{border-left:1px solid var(--qs-border);padding:16px;overflow:auto}.panel-title,.conversation-head{display:flex;align-items:center;justify-content:space-between;gap:10px;height:54px;padding:0 14px;border-bottom:1px solid var(--qs-border)}.panel-title span,.session-item small,.conversation-head span,.context-row span,.send-row span{font-size:12px;color:var(--qs-muted)}.session-list{flex:1}.session-item{display:flex;width:100%;flex-direction:column;gap:5px;padding:12px 14px;border:0;border-bottom:1px solid #edf0f3;background:transparent;text-align:left;cursor:pointer}.session-item:hover,.session-item.active{background:#e8f5f2}.session-item.active{box-shadow:inset 3px 0 var(--qs-primary)}.session-title{overflow:hidden;text-overflow:ellipsis;white-space:nowrap;font-weight:600;color:var(--qs-text)}.session-actions{display:flex;gap:6px;padding:10px;border-top:1px solid var(--qs-border)}.conversation-panel{display:flex;min-width:0;flex:1;flex-direction:column}.conversation-head>div{display:flex;min-width:0;align-items:center;gap:10px}.conversation-head strong{overflow:hidden;text-overflow:ellipsis;white-space:nowrap}.session-mobile{display:none;flex:0 0 auto}.online{color:#078a72!important}.offline{color:#d97706!important}.message-list{flex:1;padding:20px}.message{width:min(780px,88%);margin:0 0 16px;padding:13px 15px;border:1px solid var(--qs-border);border-radius:6px;background:#fff}.message.user{margin-left:auto;background:#eaf7f4;border-color:#bce4db}.message header,.message footer{display:flex;align-items:center;justify-content:space-between;gap:12px;font-size:12px;color:var(--qs-muted)}.message-content{padding:9px 0;white-space:pre-wrap;overflow-wrap:anywhere;line-height:1.7}.message footer{justify-content:flex-start}.thinking{display:flex;align-items:center;gap:6px;color:var(--qs-muted);font-size:13px}.thinking span{width:6px;height:6px;border-radius:50%;background:var(--qs-primary);animation:pulse 1.2s infinite}.thinking span:nth-child(2){animation-delay:.2s}.thinking span:nth-child(3){animation-delay:.4s}.composer{padding:12px 14px;border-top:1px solid var(--qs-border);background:#fff}.context-row,.send-row{display:flex;align-items:center;justify-content:space-between;gap:12px;margin-bottom:9px}.context-row .el-input{width:160px}.send-row{margin:9px 0 0}.evidence-mobile{display:none}.drawer-session-list{border-top:1px solid var(--qs-border)}.drawer-session-actions{display:flex;flex-wrap:wrap;gap:8px;padding-top:16px}@keyframes pulse{50%{opacity:.25}}@media(max-width:1100px){.chat-shell{grid-template-columns:210px minmax(0,1fr)}.evidence-panel{display:none}.evidence-mobile{display:inline-flex}}@media(max-width:720px){.chat-shell{display:flex;height:calc(100dvh - 220px);min-height:560px}.session-panel{display:none}.session-mobile{display:inline-flex}.message-list{padding:12px}.message{width:94%}.context-row{align-items:flex-start;flex-direction:column}.context-row .el-input{width:100%}.send-row span{display:none}}
</style>
