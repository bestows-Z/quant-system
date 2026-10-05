import { defineStore } from 'pinia'
import { taskApi } from '@/api/task'
import type { SystemTask } from '@/types/task'

export const useTaskStore = defineStore('task', {
  state: () => ({
    runningTasks: [] as SystemTask[],
    latestTasks: [] as SystemTask[]
  }),
  actions: {
    async fetchLatest() {
      this.latestTasks = await taskApi.list({ limit: 20 })
      this.runningTasks = this.latestTasks.filter((item) => ['PENDING', 'RUNNING'].includes(String(item.status)))
    }
  }
})
