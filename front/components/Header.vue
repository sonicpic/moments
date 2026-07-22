<template>
  <div
    v-if="$route.path !== '/new' && $route.path.indexOf('/edit/') < 0"
    class="header relative mb-14"
  >
    <div
      v-if="$route.path !== '/' && $route.path.indexOf('/memo/') < 0"
      :class="{ 'bg-[#4c4c4c]/80 z-10': y > 100 }"
      :style="{ width: isMobileUserAgent ? '100%' : undefined }"
      class="flex fixed justify-between items-center p-4 w-full md:w-[567px] text-white top-0"
    >
      <NuxtLink class="flex items-center" title="返回主页">
        <UIcon
          @click="navigateTo('/')"
          name="i-carbon-chevron-left"
          class="w-5 h-5 cursor-pointer mr-4"
        />
        <span v-if="$route.path === '/user/calendar'">日历检索</span>
        <span v-else-if="$route.path === '/sys/settings'">系统设置</span>
        <span v-else-if="$route.path === '/user/settings'">用户中心</span>
        <span v-else-if="$route.path.indexOf('/tags/') >= 0">
          {{ route.params.tag || "话题专栏" }}
        </span>
        <span v-else-if="$route.path === '/friend'">友情链接</span>
        <span v-else>
          <span v-if="!global.userinfo.token && $route.path === '/user/login'">
            登录
          </span>
          <span
            v-else-if="!global.userinfo.token && $route.path === '/user/reg'"
          >
            注册
          </span>
          <span v-else>{{ props.user.nickname }} 的空间</span>
        </span>
      </NuxtLink>
      <NuxtLink
        v-if="$route.path === '/user/settings' && global.userinfo.token"
        class="hidden sm:flex"
        title="登出"
        @click="logout"
      >
        <UIcon name="i-carbon-logout" class="w-5 h-5 cursor-pointer" />
      </NuxtLink>
      <span
        v-if="$route.path === '/friend' && global.userinfo.id === 1"
        class="flex"
      >
        <UIcon
          name="i-carbon-add"
          class="w-6 h-6 cursor-pointer"
          @click="$emit('add-friend')"
        />
      </span>
    </div>

    <div
      v-if="!isMobileUserAgent"
      class="dark:bg-neutral-800 flex absolute -right-10 rounded p-2 flex-col w-fit justify-end shadow top-0 gap-2 bg-white"
    >
      <svg
        v-if="!sysConfig.hideColorMode && mode.value === 'light'"
        class="lucide lucide-moon-star-icon cursor-pointer"
        @click="toggleMode"
        xmlns="http://www.w3.org/2000/svg"
        width="20"
        height="20"
        viewBox="0 0 24 24"
        fill="none"
        stroke="#FDE047"
        stroke-width="2"
        stroke-linecap="round"
        stroke-linejoin="round"
      >
        <path d="M12 3a6 6 0 0 0 9 9 9 9 0 1 1-9-9"></path>
        <path d="M20 3v4"></path>
        <path d="M22 5h-4"></path>
      </svg>

      <svg
        v-else-if="!sysConfig.hideColorMode"
        class="lucide lucide-sun-icon cursor-pointer"
        @click="toggleMode"
        xmlns="http://www.w3.org/2000/svg"
        width="20"
        height="20"
        viewBox="0 0 24 24"
        fill="none"
        stroke="#FDE047"
        stroke-width="2"
        stroke-linecap="round"
        stroke-linejoin="round"
      >
        <circle cx="12" cy="12" r="4"></circle>
        <path d="M12 2v2"></path>
        <path d="M12 20v2"></path>
        <path d="m4.93 4.93 1.41 1.41"></path>
        <path d="m17.66 17.66 1.41 1.41"></path>
        <path d="M2 12h2"></path>
        <path d="M20 12h2"></path>
        <path d="m6.34 17.66-1.41 1.41"></path>
        <path d="m19.07 4.93-1.41 1.41"></path>
      </svg>

      <NuxtLink v-if="global.userinfo.token" to="/new" title="发表">
        <UIcon
          name="i-carbon-camera"
          class="text-[#9fc84a] w-5 h-5 cursor-pointer"
        />
      </NuxtLink>
      <NuxtLink
        v-if="$route.path !== '/user/calendar' && global.userinfo.token"
        to="/user/calendar"
        title="日历检索"
      >
        <UIcon
          name="i-jam-search-folder"
          class="text-[#9fc84a] w-5 h-5 cursor-pointer"
        />
      </NuxtLink>
      <NuxtLink v-if="!sysConfig.hideFriendLink && $route.path === '/'" to="/friend" title="友情链接">
        <UIcon
          name="i-carbon-friendship"
          class="text-[#9fc84a] w-5 h-5 cursor-pointer"
        />
      </NuxtLink>
      <NuxtLink
        v-if="$route.path !== '/sys/settings' && global.userinfo.id === 1"
        to="/sys/settings"
        title="系统设置"
      >
        <UIcon
          name="i-carbon-settings"
          class="text-[#9fc84a] w-5 h-5 cursor-pointer"
        />
      </NuxtLink>
      <NuxtLink
        v-if="$route.path !== '/user/settings' && global.userinfo.token"
        to="/user/settings"
        title="用户中心"
      >
        <UIcon
          name="i-carbon-user-avatar"
          class="text-[#9fc84a] w-5 h-5 cursor-pointer"
        />
      </NuxtLink>
      <NuxtLink v-if="!sysConfig.hideMobileLogin && !global.userinfo.token" to="/user/login" title="登录">
        <UIcon
          name="i-carbon-login"
          class="text-[#9fc84a] w-5 h-5 cursor-pointer"
        />
      </NuxtLink>
    </div>

    <div v-if="isMobileUserAgent" class="fixed top-3 right-3 z-30 flex items-center gap-2">
      <NuxtLink
        v-if="global.userinfo.token && $route.path === '/'"
        to="/new"
        title="发表"
        class="flex h-9 w-9 items-center justify-center rounded-full bg-black/35 text-white backdrop-blur"
      >
        <UIcon name="i-carbon-camera" class="h-5 w-5" />
      </NuxtLink>
      <NuxtLink
        v-if="!sysConfig.hideMobileLogin && !global.userinfo.token && $route.path === '/'"
        to="/user/login"
        title="登录"
        class="flex h-9 w-9 items-center justify-center rounded-full bg-black/35 text-white backdrop-blur"
      >
        <UIcon name="i-carbon-login" class="h-5 w-5" />
      </NuxtLink>
      <button
        type="button"
        title="菜单"
        aria-label="打开菜单"
        class="flex h-9 w-9 items-center justify-center rounded-full bg-black/35 text-white backdrop-blur"
        @click="mobileNavOpen = true"
      >
        <UIcon name="i-icon-park-solid-more-four" class="h-5 w-5" />
      </button>
    </div>

    <img
      :class="sysConfig.coverDescription ? 'cursor-pointer' : ''"
      class="header-img w-full"
      :src="props.user.coverUrl"
      alt=""
      @click="showCoverDescription"
    />
    <Transition name="cover-tip">
      <div
        v-if="coverTip"
        :style="{ left: `${coverTip.x}px`, top: `${coverTip.y}px` }"
        class="pointer-events-none absolute z-20 max-w-[min(18rem,calc(100%-2rem))] -translate-x-1/2 -translate-y-1/2 rounded-2xl bg-black/70 px-3 py-2 text-xs leading-5 text-white shadow-lg backdrop-blur"
      >
        {{ coverTip.text }}
      </div>
    </Transition>
    <div class="absolute right-2 bottom-[-40px]">
      <div class="userinfo flex flex-col">
        <div class="flex flex-row items-center gap-4 justify-end">
          <button
            type="button"
            :title="sysConfig.enablePinnedMemoLink ? '打开置顶内容' : undefined"
            :class="sysConfig.enablePinnedMemoLink ? 'cursor-pointer' : ''"
            class="username text-lg font-bold text-white"
            @click="openPinnedMemo"
          >
            {{ props.user.nickname }}
          </button>
          <button
            type="button"
            :title="sysConfig.enablePinnedMemoLink ? '打开置顶内容' : undefined"
            :class="sysConfig.enablePinnedMemoLink ? 'cursor-pointer' : ''"
            class="shrink-0"
            @click="openPinnedMemo"
          >
            <img
              :src="props.user.avatarUrl"
              class="avatar w-[70px] h-[70px] rounded-xl"
            />
          </button>
        </div>
        <div
          :class="!global.userinfo.token ? 'cursor-pointer select-none' : ''"
          class="slogon text-gray truncate w-full text-end text-xs mt-2"
          @dblclick="openLogin"
        >
          {{ props.user.slogan }}
        </div>
      </div>
    </div>
  </div>
</template>
<script setup lang="ts">
import { toast } from "vue-sonner";
import type { SysConfigVO, UserVO } from "~/types";
import { useGlobalState } from "~/store";

const global = useGlobalState();
const route = useRoute();
const mobileNavOpen = useState<boolean>("sidebarOpen", () => false);
const isMobileUserAgent = useMobileUserAgent();
const sysConfig = useState<SysConfigVO>("sysConfig");

const props = defineProps<{ user: UserVO }>();
const mode = useColorMode();
const { y } = useWindowScroll();
const coverTip = ref<{ x: number; y: number; text: string }>();
let coverTipTimer: ReturnType<typeof setTimeout> | undefined;

const logout = async () => {
  global.value.userinfo = {};
  await navigateTo("/");
};

const openLogin = async () => {
  if (!global.value.userinfo.token) {
    await navigateTo("/user/login");
  }
};

const openPinnedMemo = async () => {
  if (!sysConfig.value.enablePinnedMemoLink) {
    return;
  }
  try {
    const result = await useMyFetch<{ list: Array<{ id: number; pinned: boolean; externalUrl?: string }> }>("/memo/list", { page: 1, size: 10 });
    const pinnedMemo = result?.list?.find((memo) => memo.pinned);
    if (!pinnedMemo) {
      toast.error("暂未设置置顶朋友圈");
      return;
    }
    if (pinnedMemo.externalUrl) {
      window.location.assign(pinnedMemo.externalUrl);
      return;
    }
    await navigateTo(`/memo/${pinnedMemo.id}`);
  } catch {
    toast.error("暂时无法打开置顶内容");
  }
};

const toggleMode = () => {
  if (mode.preference === "system") {
    mode.preference = "dark";
  } else if (mode.preference === "dark") {
    mode.preference = "light";
  } else {
    mode.preference = "system";
    toast.success("显示模式将跟随系统设置");
  }
};

const showCoverDescription = (event: MouseEvent) => {
  if (coverTip.value) {
    coverTip.value = undefined;
    if (coverTipTimer) {
      clearTimeout(coverTipTimer);
      coverTipTimer = undefined;
    }
    return;
  }
  const description = sysConfig.value.coverDescription?.trim();
  if (!description) {
    return;
  }
  const target = event.currentTarget as HTMLImageElement;
  const rect = target.getBoundingClientRect();
  coverTip.value = {
    x: event.clientX - rect.left,
    y: event.clientY - rect.top,
    text: description,
  };
  if (coverTipTimer) {
    clearTimeout(coverTipTimer);
  }
  coverTipTimer = setTimeout(() => {
    coverTip.value = undefined;
  }, 2600);
};

onBeforeUnmount(() => {
  if (coverTipTimer) {
    clearTimeout(coverTipTimer);
  }
});
</script>

<style scoped>
.cover-tip-enter-active,
.cover-tip-leave-active {
  transition: opacity 0.18s ease, transform 0.18s ease;
}

.cover-tip-enter-from,
.cover-tip-leave-to {
  opacity: 0;
  transform: translate(-50%, -35%) scale(0.96);
}
</style>
