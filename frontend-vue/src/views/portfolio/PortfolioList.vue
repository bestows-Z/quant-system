<template>
  <PageContainer title="真实持仓" description="手动维护或导入个人持仓，不连接券商账户">
    <template #actions><el-button type="primary" @click="createVisible = true">新建组合</el-button></template>
    <el-alert title="真实持仓仅用于本地量化分析；系统不会查询券商账户或执行真实交易" type="info" show-icon class="notice" />
    <section class="data-table-card">
      <div class="table-scroll-x"><el-table :data="items" v-loading="loading" empty-text="尚未创建真实持仓组合" class="portfolio-table">
        <el-table-column prop="name" label="组合名称" min-width="180" />
        <el-table-column prop="description" label="说明" min-width="240" />
        <el-table-column prop="base_currency" label="币种" width="90" />
        <el-table-column prop="status" label="状态" width="100" />
        <el-table-column label="更新时间" width="190"><template #default="{ row }">{{ formatTime(row.updated_at) }}</template></el-table-column>
        <el-table-column label="操作" width="260" fixed="right"><template #default="{ row }"><el-button size="small" @click="$router.push(`/portfolio/${row.id}`)">持仓管理</el-button><el-button size="small" type="primary" @click="$router.push(`/portfolio/${row.id}/analysis`)">持仓分析</el-button><el-popconfirm title="确认删除该组合和持仓？" @confirm="remove(row.id)"><template #reference><el-button size="small" type="danger">删除</el-button></template></el-popconfirm></template></el-table-column>
      </el-table></div>
    </section>
    <el-dialog v-model="createVisible" title="新建真实持仓组合" width="min(480px, 92vw)">
      <el-form label-position="top"><el-form-item label="组合名称"><el-input v-model="form.name" maxlength="80" show-word-limit /></el-form-item><el-form-item label="说明"><el-input v-model="form.description" type="textarea" :rows="3" maxlength="500" show-word-limit /></el-form-item></el-form>
      <template #footer><el-button @click="createVisible=false">取消</el-button><el-button type="primary" :loading="saving" @click="create">创建</el-button></template>
    </el-dialog>
  </PageContainer>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import PageContainer from '@/components/common/PageContainer.vue'
import { portfolioApi } from '@/api/portfolio'
import type { RealPortfolio } from '@/types/portfolio'
import { formatTime } from '@/utils/time'
const items=ref<RealPortfolio[]>([]),loading=ref(false),saving=ref(false),createVisible=ref(false)
const form=reactive({name:'',description:''})
const load=async()=>{loading.value=true;try{items.value=await portfolioApi.list()}finally{loading.value=false}}
const create=async()=>{if(!form.name.trim()){ElMessage.warning('请输入组合名称');return};saving.value=true;try{await portfolioApi.create({name:form.name.trim(),description:form.description.trim()});ElMessage.success('组合已创建');createVisible.value=false;form.name='';form.description='';await load()}finally{saving.value=false}}
const remove=async(id:number)=>{await portfolioApi.remove(id);ElMessage.success('组合已删除');await load()}
onMounted(load)
</script>
<style scoped>.notice{margin-bottom:14px}.portfolio-table{min-width:960px}</style>
