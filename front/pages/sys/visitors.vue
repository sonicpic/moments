<template>
  <Header v-if="isAdmin" :user="currentUser" />

  <main v-if="isAdmin" class="my-4 space-y-4 p-4 dark:bg-neutral-800">
    <div class="flex items-start justify-between gap-3 px-1">
      <div>
        <h1 class="text-xl font-semibold text-neutral-800 dark:text-white">访客记录</h1>
        <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">共 {{ total }} 位访客，按最近访问时间排序。</p>
      </div>
      <UButton size="sm" variant="soft" icon="i-carbon-renew" :loading="loading" @click="reload">刷新</UButton>
    </div>

    <section class="overflow-hidden rounded-xl border border-gray-100 bg-white shadow-sm dark:border-neutral-700 dark:bg-neutral-900">
      <div v-if="!loading && !visitors.length" class="p-8 text-center text-sm text-gray-400">暂无访客记录</div>

      <div class="hidden overflow-x-auto md:block">
        <table class="min-w-full text-left text-sm">
          <thead class="bg-gray-50 text-xs text-gray-500 dark:bg-neutral-800 dark:text-gray-300">
            <tr>
              <th class="px-4 py-3 font-medium">IP 地址</th>
              <th class="px-4 py-3 font-medium">设备</th>
              <th class="px-4 py-3 font-medium">浏览器</th>
              <th class="px-4 py-3 font-medium">系统</th>
              <th class="px-4 py-3 text-center font-medium">访问</th>
              <th class="px-4 py-3 font-medium">最近访问</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-gray-100 dark:divide-neutral-800">
            <tr v-for="visitor in visitors" :key="visitor.id" class="text-gray-700 dark:text-gray-200">
              <td class="whitespace-nowrap px-4 py-3 font-mono text-xs">{{ visitor.ipAddress || "-" }}</td>
              <td class="px-4 py-3"><p>{{ deviceLabel(visitor) }}</p><p class="mt-1 text-xs text-gray-400">{{ visitor.deviceType || "未知" }}</p></td>
              <td class="px-4 py-3">{{ browserLabel(visitor) }}</td>
              <td class="px-4 py-3">{{ osLabel(visitor) }}</td>
              <td class="px-4 py-3 text-center">{{ visitor.visitCount }}</td>
              <td class="whitespace-nowrap px-4 py-3 text-xs">{{ formatTime(visitor.lastSeenAt) }}</td>
            </tr>
          </tbody>
        </table>
      </div>

      <div class="space-y-3 p-3 md:hidden">
        <article v-for="visitor in visitors" :key="visitor.id" class="rounded-lg bg-gray-50 p-3 text-sm dark:bg-neutral-800">
          <div class="flex items-start justify-between gap-2"><span class="font-mono text-xs text-gray-600 dark:text-gray-300">{{ visitor.ipAddress || "-" }}</span><span class="text-xs text-gray-400">{{ visitor.visitCount }} 次访问</span></div>
          <p class="mt-2 font-medium">{{ deviceLabel(visitor) }} · {{ browserLabel(visitor) }}</p>
          <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ osLabel(visitor) }} · 最近 {{ formatTime(visitor.lastSeenAt) }}</p>
          <details v-if="visitor.userAgent" class="mt-2 text-xs text-gray-400"><summary class="cursor-pointer">查看 User-Agent</summary><p class="mt-1 break-all">{{ visitor.userAgent }}</p></details>
        </article>
      </div>
    </section>

    <div class="flex items-center justify-between px-1 text-sm text-gray-500 dark:text-gray-300">
      <UButton size="xs" color="gray" variant="soft" :disabled="page <= 1 || loading" @click="previousPage">上一页</UButton>
      <span>第 {{ page }} 页</span>
      <UButton size="xs" color="gray" variant="soft" :disabled="!hasNext || loading" @click="nextPage">下一页</UButton>
    </div>
  </main>
</template>

<script setup lang="ts">
import type { UserVO, VisitorVO } from "~/types";
import { toast } from "vue-sonner";
import { useGlobalState } from "~/store";

const global = useGlobalState();
const currentUser = useState<UserVO>("userinfo");
const isAdmin = computed(() => global.value.userinfo.id === 1);
const visitors = ref<VisitorVO[]>([]);
const total = ref(0);
const hasNext = ref(false);
const page = ref(1);
const loading = ref(false);

const formatTime = (value?: string) => {
  if (!value) return "-";
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? "-" : new Intl.DateTimeFormat("zh-CN", { dateStyle: "short", timeStyle: "short" }).format(date);
};
const deviceLabel = (visitor: VisitorVO) => visitor.deviceModel || (visitor.deviceType === "desktop" ? "桌面设备" : visitor.deviceType || "未知设备");
const browserLabel = (visitor: VisitorVO) => [visitor.browser, visitor.browserVer].filter(Boolean).join(" ") || "未知浏览器";
const osLabel = (visitor: VisitorVO) => [visitor.os, visitor.osVersion].filter(Boolean).join(" ") || "未知系统";

const reload = async () => {
  if (!isAdmin.value) return;
  loading.value = true;
  try {
    const result = await useMyFetch<{ list: VisitorVO[]; total: number; hasNext: boolean }>("/visitor/list", { page: page.value, size: 30 });
    visitors.value = result.list;
    total.value = result.total;
    hasNext.value = result.hasNext;
  } catch (error) {
    toast.error(error instanceof Error ? error.message : "读取访客记录失败");
  } finally {
    loading.value = false;
  }
};

const previousPage = async () => {
  if (page.value <= 1) return;
  page.value--;
  await reload();
};
const nextPage = async () => {
  if (!hasNext.value) return;
  page.value++;
  await reload();
};

onMounted(async () => {
  if (!isAdmin.value) {
    await navigateTo("/");
    return;
  }
  await reload();
});
</script>
