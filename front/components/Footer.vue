<template>
  <footer class="footer px-4 py-6 text-center text-xs leading-6 text-gray-400 dark:text-gray-500">
    <p>
      Powered by
      <a
        class="transition hover:text-[#719832]"
        href="https://github.com/kingwrcy/moments"
        target="_blank"
        rel="noreferrer"
      >Moments</a>
    </p>
    <p>Copyright © {{ year }} 大潘</p>
    <a
      v-if="sysConfig.beiAnNo"
      class="transition hover:text-[#719832]"
      href="https://beian.miit.gov.cn/"
      target="_blank"
      rel="noreferrer"
    >
      {{ sysConfig.beiAnNo }}
    </a>
  </footer>
</template>

<script setup lang="ts">
import type { SysConfigVO } from "~/types";

const sysConfig = useState<SysConfigVO>("sysConfig");
const year = new Date().getFullYear();
</script>
