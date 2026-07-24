<template>
  <div class="dashboard-container">
    <el-card class="welcome-card">
      <h3>欢迎使用管理面板</h3>
      <p>当前登录用户: <strong>{{ username }} </strong>，欢迎您</p>
      <el-divider />
      <div class="stats-section">
        <el-row :gutter="20">
          <el-col :xs="24" :sm="12" :lg="8">
            <el-statistic title="友链总数" :value="stats.status_data.friend_link_count" />
          </el-col>
          <el-col :xs="24" :sm="12" :lg="8">
            <el-statistic title="RSS文章总数" :value="stats.status_data.rss_post_count" />
          </el-col>
          <el-col :xs="24" :sm="12" :lg="8">
            <el-statistic title="在线时长" :value="stats.uptime" />
          </el-col>
          <el-col :xs="24" :sm="12" :lg="8">
            <el-statistic title="数据库占用" :value="formatBytes(stats.database_size_bytes)" />
          </el-col>
          <el-col :xs="24" :sm="12" :lg="8">
            <el-statistic title="数据目录占用" :value="formatBytes(stats.data_folder_size_bytes)" />
          </el-col>
        </el-row>
      </div>
    </el-card>
    <el-card class="chart-card">
      <el-row :gutter="20">
        <el-col :span="12">
          <div ref="pieChart" style="width: 100%; height: 400px"></div>
        </el-col>
        <el-col :span="12">
          <div ref="lineChart" style="width: 100%; height: 400px"></div>
        </el-col>
      </el-row>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { statsApi } from '@/api/stats'
import type { SystemStatus } from '@/model/stats'
import * as echarts from 'echarts/core'
import { LineChart, PieChart } from 'echarts/charts'
import {
  GridComponent,
  LegendComponent,
  TitleComponent,
  TooltipComponent
} from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'

echarts.use([
  LineChart,
  PieChart,
  GridComponent,
  LegendComponent,
  TitleComponent,
  TooltipComponent,
  CanvasRenderer
])

const username = ref('')
const pieChart = ref<HTMLElement | null>(null)
const lineChart = ref<HTMLElement | null>(null)

const stats = ref<SystemStatus>({
  uptime: '0s',
  database_size_bytes: 0,
  data_folder_size_bytes: 0,
  status_data: {
    friend_link_count: 0,
    rss_count: 0,
    rss_post_count: 0,
    friend_link_status_pie: [],
    rss_post_count_monthly: []
  },
  time: 0
})

const formatBytes = (bytes: number): string => {
  if (!Number.isFinite(bytes) || bytes <= 0) {
    return '0 B'
  }

  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const unitIndex = Math.min(Math.floor(Math.log(bytes) / Math.log(1024)), units.length - 1)
  const value = bytes / 1024 ** unitIndex
  const fractionDigits = unitIndex === 0 || value >= 100 ? 0 : value >= 10 ? 1 : 2

  return `${value.toFixed(fractionDigits)} ${units[unitIndex]}`
}

onMounted(async () => {
  username.value = localStorage.getItem('username') || '管理员'
  try {
    const res = await statsApi.getSystemStatus()
    if (res.code === 200) {
      stats.value = res.data
      initCharts()
    } else {
      ElMessage.error(res.message || '获取状态信息失败')
    }
  } catch (error) {
    ElMessage.error('请求状态信息时出错')
  }
})

const initCharts = () => {
  if (pieChart.value) {
    const pie = echarts.init(pieChart.value)
    pie.setOption({
      title: {
        text: '友链存活状态',
        left: 'center'
      },
      tooltip: {
        trigger: 'item'
      },
      legend: {
        orient: 'vertical',
        left: 'left'
      },
      series: [
        {
          name: '友链状态',
          type: 'pie',
          radius: '50%',
          data: stats.value.status_data.friend_link_status_pie.map((item) => ({
            value: item.count,
            name: item.status
          })),
          emphasis: {
            itemStyle: {
              shadowBlur: 10,
              shadowOffsetX: 0,
              shadowColor: 'rgba(0, 0, 0, 0.5)'
            }
          }
        }
      ]
    })
  }

  if (lineChart.value) {
    const line = echarts.init(lineChart.value)
    line.setOption({
      title: {
        text: '每月 Feed Post 数量',
        left: 'center'
      },
      tooltip: {
        trigger: 'axis'
      },
      xAxis: {
        type: 'category',
        data: stats.value.status_data.rss_post_count_monthly.map((item) => item.month)
      },
      yAxis: {
        type: 'value'
      },
      series: [
        {
          name: '文章数量',
          type: 'line',
          data: stats.value.status_data.rss_post_count_monthly.map((item) => item.count)
        }
      ]
    })
  }
}
</script>

<style scoped>
.dashboard-container {
  padding: 20px;
}

.welcome-card,
.chart-card {
  max-width: 1200px;
  margin: 0 auto 20px auto;
}

.welcome-card h3 {
  margin: 0 0 16px 0;
  color: #303133;
  font-size: 24px;
}

.welcome-card p {
  color: #606266;
  margin: 0 0 16px 0;
}

.info-section h4 {
  color: #303133;
  margin: 0 0 12px 0;
}

.info-section ul {
  margin: 0;
  padding-left: 20px;
  color: #606266;
  line-height: 1.8;
}

.stats-section {
  margin-top: 16px;
}

.stats-section :deep(.el-col) {
  margin-bottom: 20px;
}
</style>
