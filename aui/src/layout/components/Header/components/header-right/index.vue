<template>
    <div class="header_setting" :class="isMobile && 'head-absolute-fix'">
        <a-dropdown trigger="hover" @select="onLange">
            <a-button size="mini" type="text" class="icon_btn" id="system-language">
                <template #icon>
                    <icon-language :size="18" />
                </template>
            </a-button>
            <template #content>
                <a-doption :disabled="language === 'zh-CN'">{{ $t(`system.zh-CN`) }}</a-doption>
                <a-doption :disabled="language === 'en-US'">{{ $t(`system.en-US`) }}</a-doption>
            </template>
        </a-dropdown>

        <a-tooltip :content="$t(`system.${!darkMode ? 'switch-to-night-mode' : 'switch-to-daytime-mode'}`)">
            <a-button size="mini" type="text" class="icon_btn" id="system-dark" @click="onNightMode">
                <template #icon>
                    <icon-sun-fill v-if="!darkMode" :size="18" />
                    <icon-moon-fill v-else :size="18" />
                </template>
            </a-button>
        </a-tooltip>

        <a-popover position="bottom" trigger="click">
            <a-badge :count="unreadCount > 99 ? '99+' : unreadCount" :offset="[-2, 6]">
                <a-button size="mini" type="text" class="icon_btn" id="system-notice">
                    <template #icon>
                        <icon-notification :size="18" />
                    </template>
                </a-button>
            </a-badge>
            <template #content>
                <Notice />
            </template>
        </a-popover>

        <a-tooltip :content="$t(`system.${fullScreen ? 'full-screen' : 'exit-full-screen'}`)">
            <a-button size="mini" type="text" class="icon_btn" id="system-fullscreen" @click="onFullScreen">
                <template #icon>
                    <icon-fullscreen v-if="fullScreen" :size="18" />
                    <icon-fullscreen-exit v-else :size="18" />
                </template>
            </a-button>
        </a-tooltip>

        <a-tooltip :content="$t(`system.system-settings`)">
            <a-button size="mini" type="text" class="icon_btn" id="system-settings" @click="onSystemSetting">
                <template #icon>
                    <icon-settings :size="18" />
                </template>
            </a-button>
        </a-tooltip>

        <a-tooltip :content="$t(`system.theme-settings`)">
            <a-button size="mini" type="text" class="icon_btn" id="system-theme" @click="onThemeSetting">
                <template #icon>
                    <icon-skin :size="18" />
                </template>
            </a-button>
        </a-tooltip>

        <a-dropdown trigger="hover" :popup-max-height="false">
            <div class="my_setting" id="system-my-setting">
                <a-image width="32" height="32" fit="cover" :src="account.avatar" class="my_image" />
                <span class="user-nickname">{{ account.nickname }}</span>
                <div class="icon_down">
                    <icon-down style="stroke-width: 3" />
                </div>
            </div>
            <template #content>
                <a-doption @click="onPerson(1)">
                    <template #default>
                        <s-svg-icon :name="'user'" :size="18" />
                        <span class="margin-left-text">{{ $t(`system.personal-information`) }}</span>
                    </template>
                </a-doption>
                <a-doption @click="onPerson(2)">
                    <template #default>
                        <s-svg-icon :name="'lock-pwd'" :size="18" />
                        <span class="margin-left-text">{{ $t(`system.change-password`) }}</span>
                    </template>
                </a-doption>
                <a-doption v-if="showGlobalTenant" @click="switchToGlobalTenant">
                    <template #default>
                        <s-svg-icon :name="'home'" :size="18" />
                        <span class="margin-left-text">{{ $t('system.global-tenant') }}</span>
                    </template>
                </a-doption>
                <a-dropdown v-if="showTenantSwitch" trigger="hover" position="right">
                    <a-doption>
                        <template #default>
                            <s-svg-icon :name="'switch'" :size="18" />
                            <span class="margin-left-text">{{ $t('system.switch-tenant') }}</span>
                            <icon-down style="margin-left: auto; stroke-width: 3" />
                        </template>
                    </a-doption>
                    <template #content>
                        <a-doption v-for="tenant in switchableTenants" :key="tenant.id" @click="switchTenant(tenant)">
                            <template #default>
                                <span>{{ tenant.name }}</span>
                            </template>
                        </a-doption>
                    </template>
                </a-dropdown>
                <a-doption @click="onProject">
                    <template #default>
                        <s-svg-icon :name="'gitee'" :size="18" />
                        <span class="margin-left-text">{{ $t(`system.project-address`) }}</span>
                    </template>
                </a-doption>
                <a-divider margin="0" />
                <a-doption @click="logOut">
                    <template #default>
                        <s-svg-icon :name="'exit'" :size="18" />
                        <span class="margin-left-text">{{ $t(`system.logout`) }}</span>
                    </template>
                </a-doption>
            </template>
        </a-dropdown>
    </div>
    <SystemSettings :system-open="systemOpen" @system-cancel="systemOpen = false" />
    <ThemeSettings :theme-open="themeOpen" @theme-cancel="themeOpen = false" />
</template>

<script setup lang="ts">
import { computed, ref } from "vue";
import { Modal } from "@arco-design/web-vue";
import { storeToRefs } from "pinia";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { logout } from "@/api/user";
import { useDevicesSize } from "@/hooks/useDevicesSize";
import { useThemeMethods } from "@/hooks/useThemeMethods";
import Notice from "@/layout/components/Header/components/Notice/ws-notice.vue";
import SystemSettings from "@/layout/components/Header/components/system-settings/index.vue";
import ThemeSettings from "@/layout/components/Header/components/theme-settings/index.vue";
import { useNoticeStoreHook } from "@/store/modules/notice";
import { useRouteConfigStore } from "@/store/modules/route-config";
import { useThemeConfig } from "@/store/modules/theme-config";
import { useUserStoreHook } from "@/store/modules/user";

const i18n = useI18n();
const router = useRouter();
const { isMobile } = useDevicesSize();
const themeStore = useThemeConfig();
const { language, darkMode } = storeToRefs(themeStore);
const userStore = useUserStoreHook();
const { account } = storeToRefs(userStore);
const noticeStore = useNoticeStoreHook();
const unreadCount = computed(() => noticeStore.unreadCount);

const showTenantSwitch = computed(() => {
    return !!account.value.tenants && account.value.tenants.some((tenant: any) => tenant.id !== account.value.tenantID);
});

const showGlobalTenant = computed(() => {
    return (account.value.defaultTenant === null || account.value.defaultTenant === undefined) && account.value.tenantID > 0;
});

const switchableTenants = computed(() => {
    if (!account.value.tenants) return [];
    return account.value.tenants.filter((tenant: any) => tenant.id !== account.value.tenantID);
});

const switchTenant = async (tenant: any) => {
    Modal.confirm({
        title: i18n.t("system.switch-tenant-title"),
        content: i18n.t("system.switch-tenant-confirm", { name: tenant.name }),
        hideCancel: false,
        closable: true,
        onBeforeOk: async () => {
            try {
                await userStore.switchTenant(tenant.id);
                await userStore.getUserInfo();
                window.location.reload();
                return true;
            } catch (error: any) {
                console.error("切换租户失败:", error);
                return false;
            }
        }
    });
};

const switchToGlobalTenant = async () => {
    Modal.confirm({
        title: i18n.t("system.switch-tenant-title"),
        content: i18n.t("system.switch-global-tenant-confirm"),
        hideCancel: false,
        closable: true,
        onBeforeOk: async () => {
            try {
                await userStore.switchTenant(0);
                await userStore.getUserInfo();
                window.location.reload();
                return true;
            } catch (error: any) {
                console.error("切换租户失败:", error);
                return false;
            }
        }
    });
};

const systemOpen = ref(false);
const onSystemSetting = () => {
    systemOpen.value = true;
};

const themeOpen = ref(false);
const onThemeSetting = () => {
    themeOpen.value = true;
};

const fullScreen = ref(true);
const onFullScreen = () => {
    if (!document.fullscreenElement) {
        document.documentElement.requestFullscreen();
        fullScreen.value = false;
    } else if (document.exitFullscreen) {
        document.exitFullscreen();
        fullScreen.value = true;
    }
};

const onNightMode = () => {
    darkMode.value = !darkMode.value;
    const { setDarkMode } = useThemeMethods();
    setDarkMode();
};

const onLange = (value: string) => {
    if (value === "Chinese" || value === "中文") {
        themeStore.setLanguage("zh-CN");
    } else {
        themeStore.setLanguage("en-US");
    }
    i18n.locale.value = language.value;
};

const onPerson = (type: number) => {
    router.push({
        path: "/system/userinfo",
        query: {
            id: account.value.id,
            userName: account.value.username,
            type
        }
    });
};

const onProject = () => {
    window.open("https://github.com/qxkjsoft/ginfast-back", "_blank");
};

const logOut = () => {
    Modal.warning({
        title: "提示",
        content: "确定退出登录？",
        hideCancel: false,
        closable: true,
        onBeforeOk: async () => {
            try {
                await logout().catch((error: any) => {
                    if (!error.isCancelRequest) {
                        console.warn("退出登录接口调用失败，但继续执行本地清理:", error);
                    }
                });
                noticeStore.disconnect();
                await userStore.logOut();
                useRouteConfigStore().resetRoute();
                router.replace("/login");
                return true;
            } catch {
                return false;
            }
        }
    });
};
</script>

<style lang="scss" scoped>
.head-absolute-fix {
    position: absolute;
    top: 0;
    right: $padding;
}

.header_setting {
    display: flex;
    align-items: center;
    justify-content: space-between;
    height: 100%;
    background-color: $color-bg-2;

    > .icon_btn {
        box-sizing: border-box;
        display: flex;
        align-items: center;
        justify-content: space-around;
        width: $icon-box;
        height: $icon-box;
        margin-left: $margin;
        color: $color-text-1;
        border-radius: $radius-box-1;
    }

    .my_setting {
        display: flex;
        align-items: center;
        justify-content: space-between;
        height: 32px;
        margin-left: $margin;
        overflow: hidden;

        .my_image {
            margin-right: 8px;
            border-radius: 50%;
        }

        .user-nickname {
            white-space: nowrap;
        }

        .icon_down {
            margin: 0 0 0 5px;
            transform: rotate(0deg);
            transition: transform 0.2s;
        }
    }
}

:deep(.arco-dropdown-open) {
    .icon_down {
        transform: rotate(180deg) !important;
    }
}

.margin-left-text {
    margin-left: $margin-text;
}
</style>
