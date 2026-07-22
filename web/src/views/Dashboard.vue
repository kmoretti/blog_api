<template>
  <div class="dashboard-container">
    <el-card class="welcome-card">
      <h3>欢迎使用管理面板</h3>
      <p>当前登录用户: <strong>{{ username }} </strong>，欢迎您</p>
      <el-divider />
      <div class="stats-section">
        <el-row :gutter="16">
          <el-col :xs="24" :sm="24" :md="8" :lg="8">
            <el-statistic title="友链总数" :value="stats.status_data.friend_link_count" />
          </el-col>
          <el-col :xs="24" :sm="24" :md="8" :lg="8">
            <el-statistic title="RSS文章总数" :value="stats.status_data.rss_post_count" />
          </el-col>
          <el-col :xs="24" :sm="24" :md="8" :lg="8">
            <el-statistic title="在线时长" :value="uptimeSeconds" />
          </el-col>
        </el-row>
      </div>
    </el-card>
    <el-card class="chart-card">
      <el-row :gutter="16">
        <el-col :xs="24" :sm="24" :md="12" :lg="12">
          <div ref="pieChart" style="width: 100%; height: 400px" class="chart-container"></div>
        </el-col>
        <el-col :xs="24" :sm="24" :md="12" :lg="12">
          <div ref="lineChart" style="width: 100%; height: 400px" class="chart-container"></div>
        </el-col>
      </el-row>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount } from 'vue'
import { ElMessage } from 'element-plus'
import { statsApi } from '@/api/stats'
import type { SystemStatus } from '@/model/stats'
import * as echarts from 'echarts'

const username = ref('')
const pieChart = ref<HTMLElement | null>(null)
const lineChart = ref<HTMLElement | null>(null)

const stats = ref<SystemStatus>({
  uptime: '0s',
  status_data: {
    friend_link_count: 0,
    rss_count: 0,
    rss_post_count: 0,
    friend_link_status_pie: [],
    rss_post_count_monthly: []
  },
  time: 0
})

const uptimeSeconds = ref(0)

function parseUptime(uptime: string): number {
  const match = uptime.match(/(\d+)h/)
  if (match) return parseInt(match[1]) * 3600
  const matchMin = uptime.match(/(\d+)m/)
  if (matchMin) return parseInt(matchMin[1]) * 60
  const matchSec = uptime.match(/(\d+)s/)
  if (matchSec) return parseInt(matchSec[1])
  return 0
}

onMounted(async () => {
  username.value = localStorage.getItem('username') || '管理员'
  try {
    const res = await statsApi.getSystemStatus()
    if (res.code === 200) {
      stats.value = res.data
      uptimeSeconds.value = parseUptime(res.data.uptime)
      initCharts()
    } else {
      ElMessage.error(res.message || '获取状态信息失败')
    }
  } catch (error) {
    // axios 拦截器已处理错误提示，这里不再重复弹窗
  }
})

let chartInstances: echarts.ECharts[] = []
let themeObserver: MutationObserver | null = null

onMounted(() => {
  themeObserver = new MutationObserver(() => {
    // Re-render charts when theme class changes
    chartInstances.forEach((chart) => {
      const textColor = getChartTextColor()
      chart.setOption({
        backgroundColor: 'transparent',
        title: { textStyle: { color: textColor } },
        legend: { textStyle: { color: textColor } }
      })
      chart.setOption({
        xAxis: { axisLabel: { color: textColor }, axisLine: { lineStyle: { color: textColor } } },
        yAxis: { axisLabel: { color: textColor }, axisLine: { lineStyle: { color: textColor } } }
      })
    })
  })
  themeObserver.observe(document.documentElement, {
    attributes: true,
    attributeFilter: ['class']
  })
})

onBeforeUnmount(() => {
  if (themeObserver) {
    themeObserver.disconnect()
    themeObserver = null
  }
  chartInstances.forEach((chart) => chart.dispose())
  chartInstances = []
})

function getChartTextColor(): string {
  return getComputedStyle(document.documentElement).getPropertyValue('--ink').trim() || '#2c3531'
}

const initCharts = () => {
  const textColor = getChartTextColor()
  if (pieChart.value) {
    const pie = echarts.init(pieChart.value)
    pie.setOption({
      backgroundColor: 'transparent',
      title: {
        text: '友链存活状态',
        left: 'center',
        textStyle: { color: textColor }
      },
      tooltip: {
        trigger: 'item'
      },
      legend: {
        orient: 'vertical',
        left: 'left',
        textStyle: { color: textColor }
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
    chartInstances.push(line)
    line.setOption({
      backgroundColor: 'transparent',
      title: {
        text: '每月 Feed Post 数量',
        left: 'center',
        textStyle: { color: textColor }
      },
      tooltip: {
        trigger: 'axis'
      },
      xAxis: {
        type: 'category',
        data: stats.value.status_data.rss_post_count_monthly.map((item) => item.month),
        axisLabel: { color: textColor },
        axisLine: { lineStyle: { color: textColor } }
      },
      yAxis: {
        type: 'value',
        axisLabel: { color: textColor },
        axisLine: { lineStyle: { color: textColor } },
        splitLine: { lineStyle: { color: 'rgba(128,128,128,0.15)' } }
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
  color: var(--el-text-color-primary);
  font-size: 24px;
}
.welcome-card p {
  color: var(--el-text-color-regular);
  margin: 0 0 16px 0;
}
.info-section h4 {
  color: var(--el-text-color-primary);
  margin: 0 0 12px 0;
}
.info-section ul {
  margin: 0;
  padding-left: 20px;
  color: var(--el-text-color-regular);
  line-height: 1.8;
}
.stats-section {
  margin-top: 16px;
}
@media (max-width: 767px) {
  .chart-container {
    height: 250px !important;
  }
}
@media (min-width: 768px) and (max-width: 1023px) {
  .chart-container {
    height: 300px !important;
  }
}
</style>