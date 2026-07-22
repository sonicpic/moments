<template>
  <Header :user="currentUser" />

  <main class="my-4 space-y-4 p-4 dark:bg-neutral-800">
    <div class="px-1">
      <h1 class="text-xl font-semibold text-neutral-800 dark:text-white">用户中心</h1>
      <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">修改后将自动保存，无需再单独点击保存按钮。</p>
    </div>

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

    <section v-show="activeTab === 'profile'" class="space-y-4">
      <SettingsCard title="形象展示" description="头像和顶部图片会同步显示在朋友圈首页。">
        <UFormGroup label="头像" name="avatarUrl">
          <UInput type="file" size="sm" icon="i-heroicons-folder" accept="image/*" @change="uploadAvatarUrl" />
          <div class="my-2 text-xs text-gray-400">或填写在线地址</div>
          <div class="flex items-center gap-3"><UInput v-model="state.avatarUrl" /><UAvatar :src="state.avatarUrl" size="lg" /></div>
        </UFormGroup>
        <UFormGroup label="顶部图片" name="coverUrl">
          <UInput type="file" size="sm" icon="i-heroicons-folder" accept="image/*" @change="uploadCoverUrl" />
          <div class="my-2 text-xs text-gray-400">或填写在线地址</div>
          <UInput v-model="state.coverUrl" />
          <img v-if="state.coverUrl" :src="state.coverUrl" class="mt-3 max-h-48 w-full rounded-xl object-cover" alt="顶部图片预览" />
        </UFormGroup>
      </SettingsCard>

      <SettingsCard title="个人资料" description="这些内容会展示给朋友圈访客。">
        <UFormGroup label="昵称" name="nickname"><UInput v-model="state.nickname" /></UFormGroup>
        <UFormGroup label="心情状态" name="slogan"><UInput v-model="state.slogan" /></UFormGroup>
      </SettingsCard>
    </section>

    <section v-show="activeTab === 'account'" class="space-y-4">
      <SettingsCard title="账号信息" description="登录名不可修改；邮箱用于接收评论通知。">
        <UFormGroup label="登录名" name="username"><UInput v-model="state.username" disabled /></UFormGroup>
        <UFormGroup label="邮箱" name="email"><UInput v-model="state.email" type="email" placeholder="若管理员启用了邮件通知，将在收到评论时发送邮件通知" /></UFormGroup>
      </SettingsCard>

      <SettingsCard title="修改密码" description="留空不会修改密码；输入新密码后会随下一次自动保存提交。">
        <UFormGroup label="新密码" name="password"><UInput v-model="state.password" type="password" autocomplete="new-password" placeholder="留空则不修改密码" /></UFormGroup>
      </SettingsCard>
    </section>
  </main>
</template>

<script setup lang="ts">
import type { UserVO } from "~/types";
import { toast } from "vue-sonner";
import { useUpload } from "~/utils";

type Tab = "profile" | "account";
type ProfileState = UserVO & { password: string };

const currentUser = useState<UserVO>("userinfo");
const activeTab = ref<Tab>("profile");
const tabs: { value: Tab; label: string }[] = [
  { value: "profile", label: "个人资料" },
  { value: "account", label: "账号安全" },
];
const state = reactive<ProfileState>({
  id: 0,
  username: "",
  nickname: "",
  avatarUrl: "",
  slogan: "",
  coverUrl: "",
  email: "",
  password: "",
});

const initialized = ref(false);
const saving = ref(false);
const saveQueued = ref(false);
let saveTimer: ReturnType<typeof setTimeout> | undefined;

const reload = async () => {
  const res = await useMyFetch<UserVO>("/user/profile");
  if (!res) return;
  Object.assign(state, res);
  currentUser.value = res;
  await nextTick();
  initialized.value = true;
};

const persist = async () => {
  if (!initialized.value) return;
  if (saving.value) {
    saveQueued.value = true;
    return;
  }

  saving.value = true;
  const snapshot = JSON.parse(JSON.stringify(state)) as ProfileState;
  try {
    await useMyFetch("/user/saveProfile", snapshot);
    Object.assign(currentUser.value, {
      username: snapshot.username,
      nickname: snapshot.nickname,
      avatarUrl: snapshot.avatarUrl,
      slogan: snapshot.slogan,
      coverUrl: snapshot.coverUrl,
      email: snapshot.email,
    });
    if (snapshot.password && state.password === snapshot.password) {
      state.password = "";
    }
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
  if (!initialized.value) return;
  if (saveTimer) clearTimeout(saveTimer);
  saveTimer = setTimeout(() => { void persist(); }, 700);
};

watch(state, queueSave, { deep: true });

const validateImageFiles = (files: FileList) => Array.from(files).every(file => file.type.startsWith("image/"));

const uploadAvatarUrl = async (files: FileList) => {
  if (!validateImageFiles(files)) {
    toast.error("只能上传图片");
    return;
  }
  const result = await useUpload(files);
  if (result.length) {
    state.avatarUrl = result[0];
    toast.success("头像已上传，将自动保存");
  }
};

const uploadCoverUrl = async (files: FileList) => {
  if (!validateImageFiles(files)) {
    toast.error("只能上传图片");
    return;
  }
  const result = await useUpload(files);
  if (result.length) {
    state.coverUrl = result[0];
    toast.success("顶部图片已上传，将自动保存");
  }
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
