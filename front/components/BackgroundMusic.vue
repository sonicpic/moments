<template>
  <div v-if="enabled && songs.length" class="fixed bottom-5 right-4 z-40 md:bottom-6 md:right-[calc(50%-360px)]">
    <audio
      ref="audio"
      :src="currentTrack?.url"
      preload="metadata"
      @pause="playing = false"
      @playing="playing = true"
      @timeupdate="currentTime = audio?.currentTime || 0"
      @ended="playNext"
      @error="handleAudioError"
    />

    <div
      v-if="playing && !panelOpen"
      class="absolute bottom-14 right-0 w-60 rounded-xl bg-black/65 px-3 py-2 text-right text-white shadow-lg backdrop-blur"
    >
      <p class="truncate text-xs text-white/70">{{ trackTitle }}</p>
      <p class="truncate text-sm">{{ activeLyric }}</p>
    </div>

    <div
      v-if="panelOpen"
      class="absolute bottom-14 right-0 w-72 overflow-hidden rounded-2xl bg-white/95 p-3 text-neutral-800 shadow-xl ring-1 ring-black/5 backdrop-blur dark:bg-neutral-800/95 dark:text-white"
    >
      <div class="mb-2 flex items-center justify-between gap-3">
        <div class="min-w-0">
          <p class="truncate text-sm font-medium">{{ trackTitle }}</p>
          <p class="truncate text-xs text-gray-500 dark:text-gray-400">{{ activeLyric }}</p>
        </div>
        <button type="button" class="shrink-0 p-1 text-gray-500" title="关闭播放器" @click="panelOpen = false">
          <UIcon name="i-carbon-close" class="h-4 w-4" />
        </button>
      </div>

      <div class="mb-2 flex items-center justify-center gap-4">
        <button type="button" title="上一首" class="rounded-full p-1.5 hover:bg-black/5 dark:hover:bg-white/10" @click="playPrevious">
          <UIcon name="i-carbon-skip-back-filled" class="h-4 w-4" />
        </button>
        <button type="button" :title="playing ? '暂停' : '播放'" class="flex h-9 w-9 items-center justify-center rounded-full bg-[#9fc84a] text-white" @click="toggle">
          <UIcon :name="playing ? 'i-carbon-pause-filled' : 'i-carbon-play-filled'" class="h-4 w-4" />
        </button>
        <button type="button" title="下一首" class="rounded-full p-1.5 hover:bg-black/5 dark:hover:bg-white/10" @click="playNext">
          <UIcon name="i-carbon-skip-forward-filled" class="h-4 w-4" />
        </button>
      </div>

      <div v-if="songs.length > 1" class="max-h-44 space-y-1 overflow-y-auto border-t border-black/5 pt-2 dark:border-white/10">
        <button
          v-for="(track, index) in songs"
          :key="`${track.url}-${index}`"
          type="button"
          :class="index === currentIndex ? 'bg-[#9fc84a]/15 text-[#6f9630]' : 'hover:bg-black/5 dark:hover:bg-white/10'"
          class="flex w-full items-center gap-2 rounded-lg px-2 py-1.5 text-left text-sm"
          @click="selectTrack(index)"
        >
          <UIcon :name="index === currentIndex && playing ? 'i-carbon-volume-up-filled' : 'i-carbon-music'" class="h-4 w-4 shrink-0" />
          <span class="truncate">{{ displayTitle(track, index) }}</span>
        </button>
      </div>
    </div>

    <div class="flex items-center justify-end gap-2">
      <button
        type="button"
        title="播放列表与歌词"
        aria-label="播放列表与歌词"
        class="flex h-9 w-9 items-center justify-center rounded-full bg-white/90 text-[#719832] shadow-md ring-1 ring-black/5 backdrop-blur dark:bg-neutral-800/90"
        @click="panelOpen = !panelOpen"
      >
        <UIcon name="i-carbon-music" class="h-5 w-5" />
      </button>
      <button
        type="button"
        :title="playing ? `暂停${trackTitle}` : `播放${trackTitle}`"
        :aria-label="playing ? `暂停${trackTitle}` : `播放${trackTitle}`"
        class="flex h-11 w-11 items-center justify-center rounded-full bg-[#9fc84a] text-white shadow-lg ring-4 ring-white/60 transition hover:scale-105 active:scale-95 dark:ring-neutral-900/60"
        @click="toggle"
      >
        <UIcon :name="playing ? 'i-carbon-pause-filled' : 'i-carbon-play-filled'" class="h-5 w-5" />
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { BackgroundMusicVO } from "~/types";
import { toast } from "vue-sonner";

type LyricLine = {
  time: number;
  text: string;
};

const props = withDefaults(
  defineProps<{
    enabled?: boolean;
    playlist?: BackgroundMusicVO[];
    src?: string;
    title?: string;
  }>(),
  {
    enabled: false,
    playlist: () => [],
    src: "",
    title: "背景音乐",
  },
);

const audio = ref<HTMLAudioElement | null>(null);
const playing = ref(false);
const panelOpen = ref(false);
const currentIndex = ref(0);
const currentTime = ref(0);
const lyricLines = ref<LyricLine[]>([]);

const songs = computed<BackgroundMusicVO[]>(() => {
  const playlist = (props.playlist || []).filter(track => track?.url);
  if (playlist.length) {
    return playlist;
  }
  return props.src ? [{ url: props.src, title: props.title }] : [];
});

const currentTrack = computed(() => songs.value[currentIndex.value]);
const trackTitle = computed(() => displayTitle(currentTrack.value, currentIndex.value));
const activeLyric = computed(() => {
  for (let index = lyricLines.value.length - 1; index >= 0; index--) {
    if (lyricLines.value[index].time <= currentTime.value) {
      return lyricLines.value[index].text || "♪";
    }
  }
  return lyricLines.value[0]?.text || "正在播放";
});

const displayTitle = (track?: BackgroundMusicVO, index = 0) =>
  track?.title || track?.fileName?.replace(/\.[^.]+$/, "") || `音乐 ${index + 1}`;

const parseLyrics = (raw: string): LyricLine[] => {
  const result: LyricLine[] = [];
  const timestamp = /\[(\d{1,3}):(\d{2})(?:[.:](\d{1,3}))?\]/g;

  for (const row of raw.replace(/^\uFEFF/, "").split(/\r?\n/)) {
    const text = row.replace(timestamp, "").trim();
    timestamp.lastIndex = 0;
    let match: RegExpExecArray | null;
    while ((match = timestamp.exec(row)) !== null) {
      const fraction = match[3] ? Number(match[3]) / 10 ** match[3].length : 0;
      result.push({ time: Number(match[1]) * 60 + Number(match[2]) + fraction, text });
    }
  }

  return result.sort((first, second) => first.time - second.time);
};

const loadLyrics = async (track?: BackgroundMusicVO) => {
  lyricLines.value = [];
  if (!track) {
    return;
  }
  if (track.lyrics) {
    lyricLines.value = parseLyrics(track.lyrics);
    return;
  }
  if (!track.lyricsUrl) {
    return;
  }

  try {
    const response = await fetch(track.lyricsUrl);
    if (!response.ok) {
      throw new Error(`HTTP ${response.status}`);
    }
    lyricLines.value = parseLyrics(await response.text());
  } catch {
    // 外部歌词地址可能未配置 CORS；上传的歌词会优先使用已保存的内容。
  }
};

const selectTrack = async (index: number) => {
  if (!songs.value.length) {
    return;
  }
  const shouldContinue = playing.value;
  currentIndex.value = (index + songs.value.length) % songs.value.length;
  await nextTick();
  if (shouldContinue) {
    try {
      await audio.value?.play();
    } catch {
      playing.value = false;
    }
  }
};

const toggle = async () => {
  if (!audio.value) {
    return;
  }
  if (!audio.value.paused) {
    audio.value.pause();
    return;
  }
  try {
    await audio.value.play();
  } catch {
    toast.error("背景音乐无法播放，请检查音频地址");
  }
};

const playNext = async () => {
  if (songs.value.length <= 1) {
    if (audio.value) {
      audio.value.currentTime = 0;
      await audio.value.play();
    }
    return;
  }
  await selectTrack(currentIndex.value + 1);
};

const playPrevious = async () => {
  if (songs.value.length <= 1) {
    if (audio.value) {
      audio.value.currentTime = 0;
      await audio.value.play();
    }
    return;
  }
  await selectTrack(currentIndex.value - 1);
};

const handleAudioError = () => {
  playing.value = false;
  toast.error("背景音乐无法播放，请检查音频地址");
};

watch(songs, playlist => {
  if (!playlist.length) {
    currentIndex.value = 0;
    return;
  }
  if (currentIndex.value >= playlist.length) {
    currentIndex.value = 0;
  }
}, { immediate: true });

watch(currentTrack, track => {
  currentTime.value = 0;
  lyricLines.value = [];
  void loadLyrics(track);
}, { immediate: true });
</script>
