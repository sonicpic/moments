<template>
  <Header :user="currentUser" />

  <main class="my-4 space-y-4 p-4 dark:bg-neutral-800">
    <h1 class="px-1 text-xl font-semibold text-neutral-800 dark:text-white">系统设置</h1>

    <nav class="sticky top-2 z-10 -mx-1 flex gap-2 overflow-x-auto rounded-xl bg-white/90 p-1 shadow-sm backdrop-blur dark:bg-neutral-900/90">
      <button
        v-for="tab in tabs"
        :key="tab.value"
        type="button"
        :class="activeTab === tab.value ? 'bg-[#9fc84a] text-white shadow-sm' : 'text-gray-500 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-neutral-700'"
        class="shrink-0 rounded-lg px-3 py-2 text-sm transition"
        @click="activeTab = tab.value"
      >
        {{ tab.label }}
      </button>
    </nav>

    <section v-show="activeTab === 'site'" class="space-y-4">
      <SettingsCard title="站点资料" description="影响首页展示与基本站点信息。">
        <UFormGroup label="管理员账号" name="adminUserName"><UInput v-model="state.adminUserName" /></UFormGroup>
        <UFormGroup label="网站标题" name="title"><UInput v-model="state.title" /></UFormGroup>
        <UFormGroup label="Favicon" name="favicon">
          <UInput type="file" size="sm" icon="i-heroicons-folder" accept="image/*" @change="uploadFavicon" />
          <div class="my-2 text-xs text-gray-400">或填写在线地址</div>
          <div class="flex items-center gap-3"><UInput v-model="state.favicon" /><UAvatar :src="state.favicon" /></div>
        </UFormGroup>
        <UFormGroup label="背景图简介" name="coverDescription">
          <UTextarea v-model="state.coverDescription" :rows="2" placeholder="用户单击首页背景图时，会在点击处显示这段介绍。" />
        </UFormGroup>
        <UFormGroup label="备案号" name="beiAnNo"><UInput v-model="state.beiAnNo" placeholder="没有可留空" /></UFormGroup>
      </SettingsCard>

      <SettingsCard title="内容规则" description="控制内容加载与时间显示。">
        <SettingToggle v-model="state.enableExternalAccess" label="允许外部访问朋友圈" />
        <UFormGroup v-if="state.enableExternalAccess" label="外部可见起始时间" name="externalAccessStartAt" help="留空表示公开全部朋友圈；填写后仅该时间及之后的朋友圈对外可见。管理员不受此限制。">
          <div class="flex gap-2"><UInput v-model="state.externalAccessStartAt" type="datetime-local" class="flex-1" /><UButton v-if="state.externalAccessStartAt" color="gray" variant="soft" @click="state.externalAccessStartAt = ''">清空</UButton></div>
        </UFormGroup>
        <SettingToggle v-model="state.enableAutoLoadNextPage" label="首页自动加载下一页" />
        <SettingToggle v-model="state.enableRegister" label="允许新用户注册" />
        <UFormGroup label="评论最大字数" name="maxCommentLength"><UInput v-model.number="state.maxCommentLength" type="number" /></UFormGroup>
        <UFormGroup label="发言最大高度（px，0 为不限制）" name="memoMaxHeight"><UInput v-model.number="state.memoMaxHeight" type="number" /></UFormGroup>
        <UFormGroup label="评论排序" name="commentOrder"><USelectMenu v-model="state.commentOrder" :options="commentOrders" value-attribute="value" option-attribute="label" /></UFormGroup>
        <UFormGroup label="日期格式" name="timeFormat"><USelectMenu v-model="state.timeFormat" :options="timeFormats" value-attribute="value" option-attribute="label" /></UFormGroup>
      </SettingsCard>

      <SettingsCard title="访客互动" description="管理员始终可查看互动；以下开关仅控制普通访客。">
        <p class="px-1 text-sm font-medium text-neutral-700 dark:text-neutral-100">点赞</p>
        <SettingToggle v-model="state.enableLike" label="允许用户点赞" />
        <SettingToggle v-model="state.showVisitorLikeCount" :disabled="!state.enableLike" label="允许用户查看点赞数" />
        <p class="px-1 pt-2 text-sm font-medium text-neutral-700 dark:text-neutral-100">评论</p>
        <SettingToggle v-model="state.enableComment" label="允许用户评论" />
        <SettingToggle v-model="state.showVisitorComments" :disabled="!state.enableComment" label="允许用户查看评论" />
      </SettingsCard>
    </section>

    <section v-show="activeTab === 'appearance'" class="space-y-4">
      <SettingsCard title="访客按钮" description="登录按钮隐藏后，未登录访客可双击首页签名区域打开登录页。">
        <SettingToggle v-model="state.hideFriendLink" label="隐藏友情链接按钮" />
        <SettingToggle v-model="state.hideColorMode" label="隐藏深浅模式按钮" />
        <SettingToggle v-model="state.hideMobileLogin" label="隐藏所有设备的登录按钮" />
        <SettingToggle v-model="state.enablePinnedMemoLink" label="启用头像和昵称跳转" />
        <UFormGroup v-if="state.enablePinnedMemoLink" label="头像和昵称跳转链接" name="profileLinkUrl" help="填写完整网址后会直接跳转；留空则保持原有的置顶朋友圈跳转行为。">
          <UInput v-model="state.profileLinkUrl" placeholder="https://example.com" type="url" />
        </UFormGroup>
      </SettingsCard>

      <SettingsCard title="自定义扩展" description="样式和脚本会作用于整个站点，请仅粘贴可信内容。">
        <UFormGroup label="自定义 CSS" name="css"><UTextarea v-model="state.css" :rows="6" /></UFormGroup>
        <UFormGroup label="自定义 JS" name="js"><UTextarea v-model="state.js" :rows="6" /></UFormGroup>
        <UFormGroup label="自定义 RSS" name="rss"><UTextarea v-model="state.rss" :rows="2" placeholder="留空使用默认配置" /></UFormGroup>
      </SettingsCard>
    </section>

    <section v-show="activeTab === 'media'" class="space-y-4">
      <SettingsCard title="背景音乐歌单" description="音频与歌词会跟随当前文件存储方式上传。">
        <SettingToggle v-model="state.enableBackgroundMusic" label="启用背景音乐" />
        <template v-if="state.enableBackgroundMusic">
          <div class="rounded-xl border border-dashed border-gray-300 p-3 dark:border-neutral-600">
            <p class="text-xs leading-5 text-gray-500">先上传音频，再上传同名的 <code>.lrc</code> 文件即可自动关联并显示同步歌词。未启用 S3 时保存在服务器，启用 S3 时直传对象存储。</p>
            <div class="mt-3 flex flex-wrap gap-2">
              <UInput type="file" size="sm" icon="i-heroicons-musical-note" accept="audio/*,.mp3,.m4a,.aac,.ogg,.wav,.flac,.opus" multiple @change="uploadMusic" />
              <UInput type="file" size="sm" icon="i-heroicons-document-text" accept=".lrc,text/plain" multiple @change="uploadLyrics" />
              <UButton size="sm" variant="soft" icon="i-carbon-add" @click="addMusic">添加在线音乐</UButton>
            </div>
            <p v-if="musicUploading" class="mt-2 text-xs text-gray-400">正在上传文件，请稍候…</p>
          </div>
          <p v-if="!state.backgroundMusicList.length" class="text-sm text-gray-400">尚未添加音乐。</p>
          <div v-for="(music, index) in state.backgroundMusicList" :key="music.url || music.fileName || index" class="space-y-2 rounded-xl bg-gray-50 p-3 dark:bg-neutral-700/40">
            <div class="flex items-center justify-between"><span class="text-sm font-medium">第 {{ index + 1 }} 首</span><UButton size="xs" color="red" variant="ghost" icon="i-carbon-trash-can" @click="removeMusic(index)">移除</UButton></div>
            <UInput v-model="music.title" placeholder="音乐标题" />
            <UInput v-model="music.url" placeholder="音频地址（也可手动填写公开直链）" />
            <UInput v-model="music.lyricsUrl" placeholder="歌词地址（可选）" />
            <p v-if="music.lyrics" class="text-xs text-[#719832]">已读取并保存歌词，将在播放器中同步显示。</p>
          </div>
        </template>
      </SettingsCard>

      <SettingsCard title="S3 / 对象存储" description="启用后，图片、音频和歌词均通过预签名地址直传。">
        <SettingToggle v-model="state.enableS3" label="启用 S3 存储" />
        <template v-if="state.enableS3">
          <UFormGroup label="Bucket 域名（资源访问地址）"><UInput v-model="state.s3.domain" placeholder="https://img.example.com" /></UFormGroup>
          <UFormGroup label="Endpoint 地址"><UInput v-model="state.s3.endpoint" placeholder="https://endpoint.example.com" /></UFormGroup>
          <UFormGroup label="Bucket 名称"><UInput v-model="state.s3.bucket" /></UFormGroup>
          <UFormGroup label="Bucket 地区"><UInput v-model="state.s3.region" /></UFormGroup>
          <UFormGroup label="AccessKey"><UInput v-model="state.s3.accessKey" /></UFormGroup>
          <UFormGroup label="SecretKey"><UInput v-model="state.s3.secretKey" type="password" /></UFormGroup>
          <UFormGroup label="图片缩略图后缀"><UInput v-model="state.s3.thumbnailSuffix" /></UFormGroup>
        </template>
      </SettingsCard>
    </section>

    <section v-show="activeTab === 'service'" class="space-y-4">
      <SettingsCard title="Google Recaptcha" description="开启后，匿名点赞与评论需要通过人机验证。">
        <SettingToggle v-model="state.enableGoogleRecaptcha" label="启用 Google Recaptcha" />
        <template v-if="state.enableGoogleRecaptcha">
          <UFormGroup label="SiteKey"><UInput v-model="state.googleSiteKey" /></UFormGroup>
          <UFormGroup label="SecretKey"><UInput v-model="state.googleSecretKey" type="password" /></UFormGroup>
        </template>
      </SettingsCard>

      <SettingsCard title="邮件通知" description="用于发送站点邮件提醒。">
        <SettingToggle v-model="state.enableEmail" label="启用邮件通知" />
        <template v-if="state.enableEmail">
          <UFormGroup label="SMTP 服务器"><UInput v-model="state.smtpHost" placeholder="smtp.qq.com" /></UFormGroup>
          <UFormGroup label="SMTP 端口"><UInput v-model="state.smtpPort" placeholder="465" /></UFormGroup>
          <UFormGroup label="SMTP 用户名"><UInput v-model="state.smtpUsername" /></UFormGroup>
          <UFormGroup label="SMTP 密码/授权码"><UInput v-model="state.smtpPassword" type="password" /></UFormGroup>
        </template>
      </SettingsCard>

      <SettingsCard title="文件维护" description="仅清理未被内容、头像、背景音乐或歌词引用的本地文件。">
        <UButton color="red" variant="soft" @click="showCleanFileModal = true">清理未使用的本地文件</UButton>
      </SettingsCard>
    </section>
  </main>

  <UModal v-model="showCleanFileModal" :ui="{ container: 'flex items-center justify-center backdrop-blur' }">
    <div class="rounded-xl bg-white p-5 shadow-md dark:bg-neutral-800">
      <p class="text-lg font-bold">谨慎操作</p>
      <p class="mt-2 text-sm text-gray-600 dark:text-gray-300">未使用文件会移动到上传目录的 <code>removed</code> 文件夹，不会直接删除。</p>
      <div class="mt-4 flex justify-end gap-2"><UButton color="white" @click="showCleanFileModal = false">取消</UButton><UButton @click="cleanFile">确认清理</UButton></div>
    </div>
  </UModal>
</template>

<script setup lang="ts">
import type { BackgroundMusicVO, SysConfigVO, UserVO } from "~/types";
import { toast } from "vue-sonner";
import { useUpload } from "~/utils";

type Tab = "site" | "appearance" | "media" | "service";

const currentUser = useState<UserVO>("userinfo");
const sysConfig = useState<SysConfigVO>("sysConfig", () => ({}) as SysConfigVO);
const activeTab = ref<Tab>("site");
const tabs: { value: Tab; label: string }[] = [
  { value: "site", label: "站点" },
  { value: "appearance", label: "访客界面" },
  { value: "media", label: "音乐与存储" },
  { value: "service", label: "服务维护" },
];
const commentOrders = [{ label: "倒序，越晚越靠前", value: "desc" }, { label: "正序，越早越靠前", value: "asc" }];
const timeFormats = [{ label: "几分钟前", value: "timeAgo" }, { label: "固定日期时间", value: "time" }];

const state = reactive({
  enableGoogleRecaptcha: false,
  googleSiteKey: "",
  googleSecretKey: "",
  enableAutoLoadNextPage: true,
  enableLike: true,
  showVisitorLikeCount: true,
  enableComment: true,
  showVisitorComments: true,
  enableExternalAccess: true,
  externalAccessStartAt: "",
  enableRegister: true,
  enableBackgroundMusic: false,
  backgroundMusicUrl: "",
  backgroundMusicTitle: "",
  backgroundMusicList: [] as BackgroundMusicVO[],
  hideFriendLink: false,
  hideColorMode: false,
  hideMobileLogin: false,
  enablePinnedMemoLink: false,
  profileLinkUrl: "",
  coverDescription: "",
  maxCommentLength: 120,
  memoMaxHeight: 300,
  commentOrder: "desc",
  timeFormat: "timeAgo",
  adminUserName: "admin",
  title: "极简朋友圈",
  favicon: "/favicon.ico",
  beiAnNo: "",
  css: "",
  js: "",
  rss: "",
  enableS3: false,
  s3: { domain: "", bucket: "", region: "", accessKey: "", secretKey: "", endpoint: "", thumbnailSuffix: "" },
  enableEmail: false,
  smtpHost: "",
  smtpPort: "",
  smtpUsername: "",
  smtpPassword: "",
});

const showCleanFileModal = ref(false);
const musicUploading = ref(false);
const initialized = ref(false);
const saving = ref(false);
const saveQueued = ref(false);
const bulkUpdating = ref(false);
let saveTimer: ReturnType<typeof setTimeout> | undefined;

const reload = async () => {
  const res = await useMyFetch<SysConfigVO>("/sysConfig/getFull");
  if (!res) return;
  Object.assign(state, res);
  if (!Array.isArray(state.backgroundMusicList)) state.backgroundMusicList = [];
  if (!state.backgroundMusicList.length && state.backgroundMusicUrl) {
    state.backgroundMusicList.push({ url: state.backgroundMusicUrl, title: state.backgroundMusicTitle || "背景音乐" });
  }
  initialized.value = true;
};

const persist = async () => {
  if (!initialized.value || bulkUpdating.value) return;
  if (saving.value) {
    saveQueued.value = true;
    return;
  }
  saving.value = true;
  try {
    const snapshot = JSON.parse(JSON.stringify(state));
    await useMyFetch("/sysConfig/save", snapshot);
    Object.assign(sysConfig.value, snapshot);
  } catch (error) {
    toast.error(`自动保存失败：${error}`);
  } finally {
    saving.value = false;
    if (saveQueued.value) {
      saveQueued.value = false;
      queueSave();
    }
  }
};

const queueSave = () => {
  if (!initialized.value || bulkUpdating.value) return;
  if (saveTimer) clearTimeout(saveTimer);
  saveTimer = setTimeout(() => { void persist(); }, 700);
};

watch(state, queueSave, { deep: true });

watch(() => state.enableLike, (enabled) => {
  if (!enabled) state.showVisitorLikeCount = false;
});

watch(() => state.enableComment, (enabled) => {
  if (!enabled) state.showVisitorComments = false;
});

const uploadFavicon = async (files: FileList) => {
  if (Array.from(files).some(file => !file.type.startsWith("image/"))) {
    toast.error("只能上传图片");
    return;
  }
  const result = await useUpload(files);
  if (result.length) {
    state.favicon = result[0];
    toast.success("图标已上传，将自动保存");
  }
};

const filenameBase = (name: string) => name.trim().replace(/\.[^.]+$/, "").toLocaleLowerCase();
const isAudioFile = (file: File) => file.type.startsWith("audio/") || /\.(mp3|m4a|aac|ogg|wav|flac|opus)$/i.test(file.name);
const isLyricFile = (file: File) => /\.lrc$/i.test(file.name);
const addMusic = () => state.backgroundMusicList.push({ url: "", title: "" });
const removeMusic = (index: number) => state.backgroundMusicList.splice(index, 1);

const uploadMusic = async (files: FileList) => {
  const selectedFiles = Array.from(files);
  if (!selectedFiles.length) return;
  if (selectedFiles.some(file => !isAudioFile(file))) {
    toast.error("只能上传 MP3、M4A、AAC、OGG、WAV、FLAC、OPUS 等音频文件");
    return;
  }
  bulkUpdating.value = true;
  musicUploading.value = true;
  try {
    let success = 0;
    for (const file of selectedFiles) {
      const [url] = await useUpload([file]);
      if (!url) continue;
      state.backgroundMusicList.push({ url, title: filenameBase(file.name), fileName: file.name });
      success++;
    }
    if (success) toast.success(`已上传 ${success} 首音乐；可继续上传同名 .lrc 歌词文件`);
  } finally {
    bulkUpdating.value = false;
    musicUploading.value = false;
    queueSave();
  }
};

const uploadLyrics = async (files: FileList) => {
  const selectedFiles = Array.from(files);
  if (!selectedFiles.length) return;
  if (selectedFiles.some(file => !isLyricFile(file))) {
    toast.error("歌词请使用 .lrc 文件");
    return;
  }
  bulkUpdating.value = true;
  musicUploading.value = true;
  try {
    let success = 0;
    for (const file of selectedFiles) {
      const baseName = filenameBase(file.name);
      const target = state.backgroundMusicList.find(music => filenameBase(music.fileName || music.title || "") === baseName);
      if (!target) {
        toast.error(`未找到与 ${file.name} 同名的已上传音频`);
        continue;
      }
      if (file.size > 1024 * 1024) {
        toast.error(`${file.name} 超过 1MB，无法作为歌词文件保存`);
        continue;
      }
      const lyrics = await file.text();
      const [url] = await useUpload([file]);
      if (!url) continue;
      target.lyricsUrl = url;
      target.lyrics = lyrics;
      success++;
    }
    if (success) toast.success(`已自动关联 ${success} 个同名歌词文件`);
  } finally {
    bulkUpdating.value = false;
    musicUploading.value = false;
    queueSave();
  }
};

const cleanFile = async () => {
  const result = await useMyFetch<{ num: number }>("/file/clean");
  toast.success(`成功清理 ${result.num} 个未使用文件`);
  showCleanFileModal.value = false;
};

onMounted(() => { void reload(); });
onBeforeUnmount(() => {
  if (saveTimer) {
    clearTimeout(saveTimer);
    void persist();
  }
});
</script>

<style scoped>
:deep(.settings-card > * + *) { margin-top: 1rem; }
</style>
