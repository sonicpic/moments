<template>
  <Header v-if="currentUser" v-bind:user="currentUser"/>

 <div class="pb-20">
   <UCard :ui="{base:'w-4/5 mx-auto mt-20'}">

     <p class="text-center text-2xl font-sans"> 登录</p>
     <UForm class="space-y-4" size="sm" :state="state" @keyup.enter="doLogin" >
       <UFormGroup label="用户名" name="email">
         <UInput v-model="state.username" autocomplete="username"/>
       </UFormGroup>

       <UFormGroup label="密码" name="password">
         <UInput type="password" v-model="state.password" autocomplete="current-password"/>
       </UFormGroup>
       <label class="inline-flex cursor-pointer select-none items-center gap-2 text-sm text-gray-600 dark:text-gray-300">
         <input v-model="rememberPassword" type="checkbox" class="h-4 w-4 rounded border-gray-300 text-[#9fc84a] focus:ring-[#9fc84a]" />
         <span>记住密码</span>
       </label>
       <UButtonGroup size="sm">
         <UButton @click="doLogin" :disabled="pending" :loading="pending">登录</UButton>
         <UButton color="gray" v-if="sysConfig.enableRegister" variant="solid" to="/user/reg">去注册</UButton>
       </UButtonGroup>

     </UForm>

   </UCard>
 </div>
</template>

<script setup lang="ts">
import type {LoginResp, SysConfigVO, UserVO} from "~/types";
import {useGlobalState} from "~/store";
import {toast} from "vue-sonner";
const sysConfig = useState<SysConfigVO>('sysConfig')
const currentUser = useState<UserVO>('userinfo')
const global = useGlobalState()
const state = reactive({
  username: "",
  password: ""
})
const pending = ref(false)
const rememberPassword = ref(false)
const rememberedLoginStorageKey = "moments_remembered_login"

const restoreRememberedLogin = () => {
  try {
    const saved = JSON.parse(localStorage.getItem(rememberedLoginStorageKey) || "{}") as Partial<typeof state>
    if (typeof saved.username === "string" && typeof saved.password === "string" && saved.password) {
      state.username = saved.username
      state.password = saved.password
      rememberPassword.value = true
    }
  } catch {
    localStorage.removeItem(rememberedLoginStorageKey)
  }
}

const doLogin = async () => {
  pending.value = true
  let success = false
  try {
    global.value.userinfo = await useMyFetch<LoginResp>('/user/login', state)
    if (rememberPassword.value) {
      localStorage.setItem(rememberedLoginStorageKey, JSON.stringify({ username: state.username, password: state.password }))
    } else {
      localStorage.removeItem(rememberedLoginStorageKey)
    }
    toast.success("登录成功,跳转到首页...")
    success = true
  } catch (error) {
    toast.error(error instanceof Error ? error.message : "登录失败")
  } finally {
    pending.value = false
  }
  if (success) {
    location.href = '/'
  }
}

onMounted(restoreRememberedLogin)
</script>

<style scoped>

</style>
