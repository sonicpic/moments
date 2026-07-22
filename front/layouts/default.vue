<template>
  <div
    :class="isMobileUserAgent
      ? 'min-h-dvh w-full overflow-x-clip bg-white dark:bg-neutral-900 mx-auto'
      : 'min-h-dvh w-full overflow-visible bg-white dark:bg-neutral-900 md:w-[567px] md:shadow-2xl mx-auto'"
  >
    <slot />
    <Footer />
  </div>

  <BackgroundMusic
    :enabled="sysConfigVO.enableBackgroundMusic"
    :playlist="sysConfigVO.backgroundMusicList"
    :src="sysConfigVO.backgroundMusicUrl"
    :title="sysConfigVO.backgroundMusicTitle"
  />

  <div
    title="到顶部"
    v-if="!isMobileUserAgent && y > 200"
    @click="y = 0"
    class="bottom-[20%] right-[10%] lg:right-[15%] xl:right-[20%] 2xl:right-[28%] fixed flex items-center justify-center"
  >
    <UIcon
      name="i-lets-icons-expand-top-stop"
      class="w-10 h-10 text-gray-500 cursor-pointer"
    ></UIcon>
  </div>

  <div v-if="isMobileUserAgent">
    <MobileNav />
  </div>
</template>

<script lang="ts" setup>
import type { SysConfigVO, UserVO } from "~/types";

const sysConfig = useState<SysConfigVO>("sysConfig");
const currentUser = useState<UserVO>("userinfo");
const isMobileUserAgent = useMobileUserAgent();
const currentProfile = await useMyFetch<UserVO>("/user/profile");
const sysConfigVO = await useMyFetch<SysConfigVO>("/sysConfig/get");
if (sysConfigVO) {
  sysConfig.value = sysConfigVO;
}
if (currentProfile) {
  currentUser.value = currentProfile;
}
const { y } = useWindowScroll();
useHead({
  title: sysConfigVO.title,
  link: [
    {
      rel: "shortcut icon",
      type: "image/png",
      href: sysConfigVO.favicon || "/favicon.png",
    },
    {
      rel: "apple-touch-icon-precomposed",
      href: sysConfigVO.favicon || "/favicon.png",
    },
    {
      rel: "alternate",
      type: "application/rss+xml",
      title: "我的 RSS 订阅",
      href: sysConfigVO.rss || `/rss`,
    },
  ],
  style: [
    {
      innerHTML: sysConfigVO.css || "",
    },
  ],
  script: [
    {
      type: "text/javascript",
      innerHTML: sysConfigVO.js || "",
    },
  ],
});

if (sysConfigVO.enableGoogleRecaptcha) {
  useHead({
    script: [
      {
        type: "text/javascript",
        src: `https://recaptcha.net/recaptcha/api.js?render=${sysConfigVO.googleSiteKey}`,
      },
    ],
  });
}
</script>
